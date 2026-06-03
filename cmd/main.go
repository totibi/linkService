package main

import (
	"context"
	"flag"
	"fmt"
	"linkService/internal/app/domain"
	"linkService/internal/app/logging"
	"linkService/internal/app/service"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	linkServicePb "linkService/generated/go/v1"
	"linkService/internal/app/config"

	grpcGatewayRt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {

	runServerWithSwagger()
}

func runServerWithSwagger() {
	envFile := flag.String(
		"env-file",
		"",
		"path to .env file",
	)

	flag.Parse()

	cfg, cfgErr := config.Load(*envFile)
	if cfgErr != nil {
		log.Fatalf("load config: %v", cfgErr)
	}

	// для логирования SQL-запросов
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logging.GrpcLoggingInterceptor),
	)

	// пул переиспользуемых подключений к БД postgreSQL
	dbPool, dbErr := getDbPool(context.Background(), cfg.DB.Url)
	if dbErr != nil {
		panic(dbErr)
	}

	// DI из сервиса и репозитория.
	//Привязка реализации сервиса к сгенерированному GRPC LinkServer для обработки запросов
	linkRepo := domain.NewLinkRepository(dbPool)
	linkService := service.NewAppLinkService(linkRepo)
	linkServicePb.RegisterLinkServiceServer(grpcServer, linkService)

	// grpcGateway проксирует запросы с http request на grpc request
	gwMux := grpcGatewayRt.NewServeMux(
		grpcGatewayRt.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
			md := make(metadata.MD)
			if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
				md.Set("x-request-id", reqID)
			}
			return md
		}),
	)

	// Регистрируем gateway
	err := linkServicePb.RegisterLinkServiceHandlerFromEndpoint(
		context.Background(),
		gwMux,
		fmt.Sprintf(":%d", cfg.GRPC.Port),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		panic(err)
	}

	// Главные HTTP маршрутизатор запросов
	mux := http.NewServeMux()
	mux.Handle("/v1/", logging.HttpLoggingMiddleware(gwMux)) // HTTP -> GRPC эндпоинты
	mux.HandleFunc("/swagger/", swaggerHandler)              // Swagger UI
	mux.HandleFunc("/swagger/swagged.json", swaggerJSONHandler)

	// старт GRPC сервера
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		slog.Error("Failed to listen for gRPC", "error", err)
		os.Exit(1)
	}
	go func() {
		slog.Info("Starting gRPC server", "addr", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server stopped with error", "error", err)
		}
	}()

	// Запуск HTTP сервера
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: mux,
	}

	slog.Info("Server starting",
		"grpc_port", cfg.GRPC.Port,
		"http_port", cfg.HTTP.Port,
		"swagger", fmt.Sprintf("http://localhost:%d/swagger/", cfg.HTTP.Port),
	)

	// Graceful shutdown
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer linkService.Close()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		slog.Error("HTTP server shutdown failed", "error", err)
	}
	grpcServer.GracefulStop()
	slog.Info("Server stopped")

}

func swaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./generated/api.swagger.json")
}

func swaggerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/swagger" {
		http.Redirect(
			w,
			r,
			"/swagger/?url=/swagger/swagger.json",
			http.StatusMovedPermanently,
		)
		return
	}

	http.StripPrefix(
		"/swagger/",
		http.FileServer(http.Dir("./swagger")),
	).ServeHTTP(w, r)
}

func getDbPool(ctx context.Context, dbUrl string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, err
	}
	poolCfg.ConnConfig.Tracer = &logging.PgxSlogTracer{}

	return pgxpool.NewWithConfig(ctx, poolCfg)
}
