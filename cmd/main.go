package main

import (
	"context"
	"flag"
	"linkService/internal/app"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"linkService/internal/app/config"
)

func main() {

	cfg := loadConfig()

	// для логирования SQL-запросов
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	appInstance, err := app.New(
		context.Background(),
		cfg,
	)
	if err != nil {
		log.Fatal(err)
	}

	defer appInstance.Close()

	if err := appInstance.Start(); err != nil {
		log.Fatal(err)
	}

	waitShutdown(appInstance)
}

func loadConfig() *config.Config {
	envFile := flag.String("env-file", "", "path to env file")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	return cfg
}

func waitShutdown(appInstance *app.App) {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	defer cancel()

	if err := appInstance.Shutdown(shutdownCtx); err != nil {
		slog.Error(
			"shutdown failed",
			"error",
			err,
		)
	}
}
