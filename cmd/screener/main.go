package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Screener-masterskaya/Screener_backend/internal/app"
	"github.com/Screener-masterskaya/Screener_backend/internal/config"
	"github.com/Screener-masterskaya/Screener_backend/internal/logger"
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/postgres"
)

func main() {
	os.Exit(run())
}

// run initializes application dependencies, runs the application,
// and ensures resource cleanup before returning the process exit code.
func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		return 1
	}

	log, err := logger.New(os.Stdout, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init logger:", err)
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.Postgres)
	if err != nil {
		log.Error("failed to initialize PostgreSQL pool", slog.Any("error", err))
		return 1
	}
	defer pool.Close()

	app := app.NewApp(cfg, log)

	if err := app.Run(ctx); err != nil {
		log.Error("stopped with error", slog.Any("error", err))
		return 1
	}
	return 0
}
