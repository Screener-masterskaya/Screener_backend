package app

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Screener-masterskaya/Screener_backend/internal/health"
)

// newRouter builds the root HTTP handler with core middleware and versioned API routes.

// TODO: add middleware with a logger.
func newRouter(healthHandler *health.Handler) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler.Liveness)
	router.Get("/readyz", healthHandler.Readiness)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/", okHandler)
	})
	return router
}

// okHandler responds with a JSON health/status payload for smoke checks.
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
