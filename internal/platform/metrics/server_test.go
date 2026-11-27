package metrics

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerServesOnlyMetricsRoute(t *testing.T) {
	reg := prometheus.NewRegistry()
	handler := Handler(reg)

	rec := httptestRequest(t, handler, http.MethodGet, "/metrics")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = httptestRequest(t, handler, http.MethodGet, "/anything-else")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServerRunAndGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(addr, Handler(prometheus.NewRegistry()), log, 2*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Ждём поднятия сервера и проверяем, что он отдаёт метрики.
	client := &http.Client{Timeout: time.Second}
	var resp *http.Response
	var lastErr error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, lastErr = client.Get("http://" + addr + "/metrics")
		if lastErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, lastErr)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()

	select {
	case runErr := <-errCh:
		require.NoError(t, runErr)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for metrics server to stop after context cancellation")
	}
}
