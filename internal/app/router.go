package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Screener-masterskaya/Screener_backend/internal/httpmw"
)

// Probe paths. Health routes must be registered with these constants so that
// the access log filter and the routes cannot drift apart.
const (
	healthzPath = "/healthz"
	readyzPath  = "/readyz"
)

// newRouter builds the root HTTP handler with core middleware and versioned API routes.
func newRouter(log *slog.Logger) http.Handler {
	router := chi.NewRouter()

	// Order matters: the access log needs the request ID and must see the 500 from Recovery.
	// Probes are not access-logged: they run every few seconds and would be noise.
	router.Use(httpmw.RequestID, httpmw.AccessLog(log, healthzPath, readyzPath), httpmw.Recovery(log))

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
