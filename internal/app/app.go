package app

import (
	"context"
	"fmt"
	"linkService/internal/app/handler"
	"log/slog"
	"net"
	"net/http"

	grpcGatewayRt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	linkServicePb "linkService/generated/go/v1"

	"linkService/internal/app/config"
	"linkService/internal/app/domain"
	"linkService/internal/app/logging"
	"linkService/internal/app/service"
)

type App struct {
	config *config.Config

	dBPool *pgxpool.Pool

	linkService     *service.AppLinkService
	grpcLinkHandler *handler.LinkHandler
	grpcServer      *grpc.Server
	httpServer      *http.Server

	grpcListener net.Listener
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	dbPool, err := getDbPool(ctx, cfg.DB.Url)
	if err != nil {
		return nil, err
	}

	repo := domain.NewLinkRepository(dbPool)

	linkService := service.NewAppLinkService(repo)

	grpcLinkHandler := handler.NewLinkHandler(linkService)

	app := &App{
		config:          cfg,
		dBPool:          dbPool,
		linkService:     linkService,
		grpcLinkHandler: grpcLinkHandler,
	}

	if err := app.buildServers(); err != nil {
		app.Close()
		return nil, err
	}

	return app, nil
}

func (a *App) buildServers() error {
	a.grpcServer = newGrpcServer(a)

	gateway, err := newGateway(a.config)
	if err != nil {
		return err
	}

	httpMux := newHTTPMux(gateway)

	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.HTTP.Port),
		Handler: httpMux,
	}

	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf(":%d", a.config.GRPC.Port),
	)
	if err != nil {
		return err
	}

	a.grpcListener = listener

	return nil
}

func newGrpcServer(app *App) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.GrpcLoggingInterceptor,
		),
	)

	linkServicePb.RegisterLinkServiceServer(
		server,
		app.grpcLinkHandler,
	)

	return server
}

func newGateway(cfg *config.Config) (*grpcGatewayRt.ServeMux, error) {

	gw := grpcGatewayRt.NewServeMux(
		grpcGatewayRt.WithMetadata(
			func(ctx context.Context, r *http.Request) metadata.MD {

				md := make(metadata.MD)

				if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
					md.Set("x-request-id", reqID)
				}

				return md
			},
		),
	)

	err := linkServicePb.RegisterLinkServiceHandlerFromEndpoint(
		context.Background(),
		gw,
		fmt.Sprintf(":%d", cfg.GRPC.Port),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(
				insecure.NewCredentials(),
			),
		},
	)

	return gw, err
}

func newHTTPMux(gw *grpcGatewayRt.ServeMux) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle(
		"/v1/",
		logging.HttpLoggingMiddleware(gw),
	)

	handler.RegisterSwaggerRoutes(mux)

	return mux
}

func (a *App) Start() error {

	slog.Info(
		"application starting",
		"http_port",
		a.config.HTTP.Port,
		"grpc_port",
		a.config.GRPC.Port,
		"swagger_url",
		fmt.Sprintf(
			"http://localhost:%d/swagger/",
			a.config.HTTP.Port,
		),
	)

	go func() {
		if err := a.grpcServer.Serve(a.grpcListener); err != nil {
			slog.Error(
				"gRPC server stopped",
				"error",
				err,
			)
		}
	}()

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			slog.Error(
				"HTTP server stopped",
				"error",
				err,
			)
		}
	}()

	return nil
}

func (a *App) Close() {
	slog.Info("closing resources")

	if a.linkService != nil {
		a.linkService.Close()
	}

	if a.dBPool != nil {
		a.dBPool.Close()
		slog.Info("dbpool is closed")
	}
}

func (a *App) Shutdown(ctx context.Context) error {

	slog.Info("shutting down application")
	if err := a.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	a.grpcServer.GracefulStop()
	slog.Info("application stopped")
	return nil
}

func getDbPool(ctx context.Context, dbUrl string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, err
	}
	poolCfg.ConnConfig.Tracer = &logging.PgxSlogTracer{}

	return pgxpool.NewWithConfig(ctx, poolCfg)
}
