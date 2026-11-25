package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Screener-masterskaya/Screener_backend/internal/config"
	"github.com/Screener-masterskaya/Screener_backend/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newRouter(health.New(logger, time.Second))
}

func TestRouterHealthRoutes(t *testing.T) {
	handler := testRouter()

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRouterAPIv1Root(t *testing.T) {
	handler := testRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, rec.Code, http.StatusOK)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
}

func TestAppRunAndGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	cfg := &config.Config{
		LogLevel:  "info",
		LogFormat: "text",
		HTTPServer: config.HTTPServer{
			Address:         addr,
			ReadTimeout:     2 * time.Second,
			WriteTimeout:    2 * time.Second,
			IdleTimeout:     10 * time.Second,
			ShutdownTimeout: 2 * time.Second,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application := NewApp(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		errCh <- application.Run(ctx)
	}()

	// Даём серверу короткое время на старт и проверяем живой эндпоинт.
	client := &http.Client{Timeout: time.Second}
	var resp *http.Response
	var lastErr error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, lastErr = client.Get("http://" + addr + "/api/v1/")
		if lastErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, lastErr)

	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()

	select {
	case runErr := <-errCh:
		require.NoError(t, runErr)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Run to return after context cancellation")
	}

	// Run must mark the service as shutting down, so /readyz reports 503.
	rec := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
