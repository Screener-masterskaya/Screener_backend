package httpmw

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const unmatchedRoute = "unmatched"

// AccessLog writes one record per request with the method, route pattern,
// status, duration and request ID.
//
// It logs the route pattern ("/users/{id}"), never the raw path or query,
// and no headers or bodies, so secrets do not reach the logs.
// Register it after RequestID and before the panic recovery middleware.
//
// Requests whose path exactly equals one of skipPaths are not logged. Use it
// for high-frequency probes such as health checks.
func AccessLog(log *slog.Logger, skipPaths ...string) func(http.Handler) http.Handler {
	skip := make(map[string]bool, len(skipPaths))
	for _, path := range skipPaths {
		skip[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				// Nothing was written explicitly: net/http sends 200.
				status = http.StatusOK
			}

			log.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("method", r.Method),
				slog.String("route", routePattern(r)),
				slog.Int("status", status),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("request_id", RequestIDFromContext(r.Context())),
			)
		})
	}
}

// routePattern must be called after the router has handled the request.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}

	return unmatchedRoute
}
