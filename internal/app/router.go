package app

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Screener-masterskaya/Screener_backend/internal/health"
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/metrics"
)

// newRouter собирает корневой HTTP-обработчик с базовыми middleware
// и версионированными маршрутами API.
// httpMetrics инструментирует все запросы; передайте nil, чтобы отключить
// инструментацию.
//
// Эндпоинт метрик намеренно НЕ смонтирован здесь: он обслуживается на
// отдельном адресе (METRICS_ADDR), чтобы не быть доступным на публичном
// порту API.
//
// TODO: add middleware with a logger.
func newRouter(healthHandler *health.Handler, httpMetrics *metrics.HTTP) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	if httpMetrics != nil {
		router.Use(httpMetrics.Middleware)
	}
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler.Liveness)
	router.Get("/readyz", healthHandler.Readiness)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/", okHandler)
	})
	return router
}

// okHandler отвечает JSON-полезной нагрузкой health/status для smoke-проверок.
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
