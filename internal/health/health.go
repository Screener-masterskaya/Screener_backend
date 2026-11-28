// Package health serves liveness and readiness probes for the service.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Pinger checks that an external dependency is reachable. *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Check is a named dependency the service needs before it can accept traffic.
type Check struct {
	Name   string
	Pinger Pinger
}

// Handler serves /healthz and /readyz.
type Handler struct {
	log          *slog.Logger
	timeout      time.Duration
	checks       []Check
	shuttingDown atomic.Bool
}

// New returns a Handler. timeout bounds each dependency check in /readyz.
func New(log *slog.Logger, timeout time.Duration, checks ...Check) *Handler {
	return &Handler{
		log:     log,
		timeout: timeout,
		checks:  checks,
	}
}

// SetShuttingDown makes /readyz respond 503. Call it when graceful shutdown starts.
func (h *Handler) SetShuttingDown() {
	h.shuttingDown.Store(true)
}

type response struct {
	Status string `json:"status"`
	Failed string `json:"failed,omitempty"`
}

// Liveness responds 200 while the process is alive. It does not check dependencies.
func (h *Handler) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok"})
}

// Readiness responds 200 when the service is not shutting down and every dependency is reachable, 503 otherwise.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.shuttingDown.Load() {
		writeJSON(w, http.StatusServiceUnavailable, response{Status: "shutting_down"})
		return
	}

	for _, c := range h.checks {
		if err := h.ping(r.Context(), c); err != nil {
			h.log.Warn("readiness check failed", slog.String("check", c.Name), slog.String("error", err.Error()))
			writeJSON(w, http.StatusServiceUnavailable, response{Status: "not_ready", Failed: c.Name})
			return
		}
	}

	writeJSON(w, http.StatusOK, response{Status: "ready"})
}

func (h *Handler) ping(ctx context.Context, c Check) error {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	return c.Pinger.Ping(ctx)
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
