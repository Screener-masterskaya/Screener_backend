package httpmw_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/Screener-masterskaya/Screener_backend/internal/httpmw"
	"github.com/Screener-masterskaya/Screener_backend/internal/logger"
)

func TestAccessLog(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantRoute  string
		wantStatus int
	}{
		{name: "implicit 200", target: "/users/42", wantRoute: "/users/{id}", wantStatus: http.StatusOK},
		{name: "explicit status", target: "/teapot", wantRoute: "/teapot", wantStatus: http.StatusTeapot},
		{name: "unknown path", target: "/nope", wantRoute: "unmatched", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log, err := logger.New(&buf, "info", logger.FormatJSON)
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Use(httpmw.RequestID, httpmw.AccessLog(log))
			router.Get("/users/{id}", func(http.ResponseWriter, *http.Request) {})
			router.Get("/teapot", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})

			req := httptest.NewRequest(http.MethodGet, tt.target+"?token=query-secret", nil)
			req.Header.Set("Authorization", "Bearer header-secret")
			req.Header.Set(httpmw.HeaderRequestID, "req-1")
			router.ServeHTTP(httptest.NewRecorder(), req)

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			require.Len(t, lines, 1, "one request must produce one record")

			var rec map[string]any
			require.NoError(t, json.Unmarshal([]byte(lines[0]), &rec))
			require.Equal(t, http.MethodGet, rec["method"])
			require.Equal(t, tt.wantRoute, rec["route"])
			require.EqualValues(t, tt.wantStatus, rec["status"])
			require.Equal(t, "req-1", rec["request_id"])
			require.Contains(t, rec, "duration_ms")
			require.NotContains(t, lines[0], "secret")
		})
	}
}

func TestAccessLogSkipPaths(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantLog bool
	}{
		{name: "skipped path", target: "/healthz", wantLog: false},
		{name: "unknown path is still logged", target: "/nope", wantLog: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log, err := logger.New(&buf, "info", logger.FormatJSON)
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Use(httpmw.AccessLog(log, "/healthz"))
			router.Get("/healthz", func(http.ResponseWriter, *http.Request) {})

			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tt.target, nil))

			require.Equal(t, tt.wantLog, buf.Len() > 0)
		})
	}
}
