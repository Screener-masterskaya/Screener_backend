package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 10 * time.Second
	serverIdleTimeout  = 60 * time.Second
)

// Handler возвращает HTTP-обработчик, отдающий метрики в формате Prometheus
// по пути /metrics. Если gatherer равен nil, используется
// prometheus.DefaultGatherer, который включает коллекторы Go runtime
// и процесса.
func Handler(gatherer prometheus.Gatherer) http.Handler {
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	return mux
}

// Server публикует метрики Prometheus на отдельном адресе, независимом
// от публичного порта API, и поддерживает graceful shutdown.
type Server struct {
	log             *slog.Logger
	server          *http.Server
	shutdownTimeout time.Duration
}

// NewServer создаёт сервер метрик, слушающий addr и обслуживающий handler.
// shutdownTimeout ограничивает длительность graceful shutdown после отмены
// контекста запуска.
func NewServer(addr string, handler http.Handler, log *slog.Logger, shutdownTimeout time.Duration) *Server {
	return &Server{
		log: log,
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  serverReadTimeout,
			WriteTimeout: serverWriteTimeout,
			IdleTimeout:  serverIdleTimeout,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

// Run запускает сервер метрик и блокируется до фатальной ошибки сервера
// или отмены ctx. При отмене контекста сервер останавливается graceful
// в пределах настроенного таймаута, и Run возвращает nil.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info("metrics server starting", slog.String("addr", s.server.Addr))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("metrics listen and serve: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutdown signal received, stopping metrics server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			_ = s.server.Close()
			return fmt.Errorf("metrics graceful shutdown failed: %w", err)
		}

		if err := <-errCh; err != nil {
			return err
		}

		s.log.Info("metrics server stopped cleanly")
		return nil
	}
}
