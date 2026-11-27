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
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/metrics"
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/postgres"
)

const (
	lvlDebug = "debug"
	lvlInfo  = "info"
	lvlWarn  = "warn"
	lvlError = "error"
	logJSON  = "json"
	logText  = "text"
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

	log := setupLogger(cfg.LogLevel, cfg.LogFormat)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.Postgres)
	if err != nil {
		log.Error("failed to initialize PostgreSQL pool", Err(err))
		return 1
	}
	defer pool.Close()

	// Prometheus-инструментация на дефолтном реестре: метрики HTTP-запросов,
	// статистика pgxpool, а также коллекторы Go runtime и процесса,
	// которые client_golang регистрирует автоматически.
	httpMetrics := metrics.NewHTTP(nil)
	if err := metrics.RegisterPool(nil, pool); err != nil {
		log.Error("failed to register pgxpool metrics", Err(err))
		return 1
	}

	app := app.NewApp(cfg, log, httpMetrics)

	// Сервер метрик работает на собственном адресе (METRICS_ADDR), отдельном
	// от публичного порта API. Он останавливается graceful вместе с приложением.
	metricsServer := metrics.NewServer(cfg.MetricsAddr, metrics.Handler(nil), log, cfg.ShutdownTimeout)

	metricsErrCh := make(chan error, 1)
	go func() {
		metricsErrCh <- metricsServer.Run(ctx)
	}()

	appErr := app.Run(ctx)

	// Гарантируем остановку сервера метрик, даже если приложение завершилось
	// с ошибкой самостоятельно, и дожидаемся окончания его graceful shutdown.
	cancel()
	metricsErr := <-metricsErrCh

	if appErr != nil {
		log.Error("stopped with error", Err(appErr))
		return 1
	}
	if metricsErr != nil {
		log.Error("metrics server stopped with error", Err(metricsErr))
		return 1
	}
	return 0
}

// setupLogger configures and returns a structured logger for the given level and format.
// It falls back to a text debug logger when an unrecognized format is provided.
// TODO: move to a separate package with a logger
func setupLogger(logLevel, logFormat string) *slog.Logger {
	var log *slog.Logger

	switch logFormat {
	case logJSON:
		switch logLevel {
		case lvlDebug:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		case lvlInfo:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		case lvlWarn:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		case lvlError:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		default:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	case logText:
		switch logLevel {
		case lvlDebug:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		case lvlInfo:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		case lvlWarn:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		case lvlError:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		default:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	default:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}

// Err formats an error as a structured slog attribute under the "error" key.
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
