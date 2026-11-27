// Package httpmw contains the HTTP middleware shared by all routes.
package httpmw

import (
	"context"
	"crypto/rand"
	"net/http"
)

// HeaderRequestID is the header that carries the request ID in both directions.
const HeaderRequestID = "X-Request-ID"

const maxRequestIDLen = 64

type requestIDKey struct{}

// RequestID takes a valid X-Request-ID from the request or generates a new one,
// returns it in the response header and stores it in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if !validRequestID(id) {
			id = rand.Text()
		}

		w.Header().Set(HeaderRequestID, id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the ID stored by RequestID,
// or an empty string if there is none.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// validRequestID limits client input that ends up in logs and response headers:
// 1 to 64 ASCII letters, digits, '-' or '_'.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}

	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}

	return true
}
