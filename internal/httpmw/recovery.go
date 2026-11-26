package httpmw

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
)

const internalErrorBody = `{"error":"internal server error"}` + "\n"

// Recovery turns a panic in a handler into a log record with the stack trace
// and a 500 response. If the response has already started, its status cannot
// be changed, so the panic is only logged.
//
// http.ErrAbortHandler is re-panicked: it is the standard way to abort a response.
// Register Recovery after AccessLog, so the access log sees the 500.
// Panics in goroutines started by a handler are not caught.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				log.LogAttrs(r.Context(), slog.LevelError, "panic recovered",
					slog.String("panic", fmt.Sprint(rec)),
					slog.String("method", r.Method),
					slog.String("route", routePattern(r)),
					slog.String("request_id", RequestIDFromContext(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)

				if ww.Status() != 0 {
					return
				}
				ww.Header().Set("Content-Type", "application/json; charset=utf-8")
				ww.WriteHeader(http.StatusInternalServerError)
				// The client may be gone; there is nothing to do about a write error.
				_, _ = io.WriteString(ww, internalErrorBody)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
