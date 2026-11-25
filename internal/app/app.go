package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Screener-masterskaya/Screener_backend/internal/config"
	"github.com/Screener-masterskaya/Screener_backend/internal/health"
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/metrics"
)

// readinessTimeout bounds each dependency check in /readyz.
const readinessTimeout = 2 * time.Second

// App wires together configuration, structured logging, and the HTTP server lifecycle.
type App struct {
	cfg    *config.Config
	log    *slog.Logger
	server *http.Server
	health *health.Handler
}

// NewApp инициализирует экземпляр App и настраивает HTTP-сервер с таймаутами из cfg.
// httpMetrics инструментирует все запросы API; передайте nil, чтобы отключить
// инструментацию.
func NewApp(cfg *config.Config, log *slog.Logger, httpMetrics *metrics.HTTP) *App {
	healthHandler := health.New(log, readinessTimeout)
	handler := newRouter(healthHandler, httpMetrics)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &App{
		cfg:    cfg,
		log:    log,
		server: srv,
		health: healthHandler,
	}
}

// Run starts the HTTP server and blocks until a fatal server error occurs or ctx is canceled.
// When ctx is canceled, it attempts a graceful shutdown within cfg.ShutdownTimeout.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("server starting", slog.String("addr", a.cfg.Address))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.log.Info("shutdown signal received, stopping server")
		a.health.SetShuttingDown()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			_ = a.server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		if err := <-errCh; err != nil {
			return err
		}

		a.log.Info("server stopped cleanly")
		return nil
	}
}
