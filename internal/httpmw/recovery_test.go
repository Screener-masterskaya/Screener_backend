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

// newPanicRouter builds the full middleware chain around handlers that panic.
func newPanicRouter(t *testing.T, buf *bytes.Buffer) http.Handler {
	t.Helper()

	log, err := logger.New(buf, "info", logger.FormatJSON)
	require.NoError(t, err)

	router := chi.NewRouter()
	router.Use(httpmw.RequestID, httpmw.AccessLog(log), httpmw.Recovery(log))
	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	router.Get("/late-panic", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("boom after headers")
	})
	router.Get("/abort", func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})

	return router
}

func decodeRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		records = append(records, rec)
	}

	return records
}

func TestRecovery_Returns500AndLogsStack(t *testing.T) {
	var buf bytes.Buffer
	router := newPanicRouter(t, &buf)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(httpmw.HeaderRequestID, "req-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "boom", "panic details must not reach the client")

	records := decodeRecords(t, &buf)
	require.Len(t, records, 2)

	require.Equal(t, "panic recovered", records[0]["msg"])
	require.Equal(t, "boom", records[0]["panic"])
	require.Equal(t, "req-1", records[0]["request_id"])
	require.Contains(t, records[0]["stack"], "recovery_test.go")

	require.Equal(t, "http request", records[1]["msg"])
	require.EqualValues(t, http.StatusInternalServerError, records[1]["status"])
}

func TestRecovery_ServerKeepsServing(t *testing.T) {
	var buf bytes.Buffer
	server := httptest.NewServer(newPanicRouter(t, &buf))
	defer server.Close()

	for range 2 {
		resp, err := server.Client().Get(server.URL + "/panic")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}
}

func TestRecovery_KeepsStatusAlreadySent(t *testing.T) {
	var buf bytes.Buffer
	router := newPanicRouter(t, &buf)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/late-panic", nil))

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Empty(t, rec.Body.String())
	require.Contains(t, buf.String(), "boom after headers")
}

func TestRecovery_RepanicsErrAbortHandler(t *testing.T) {
	var buf bytes.Buffer
	router := newPanicRouter(t, &buf)

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
	})
}
