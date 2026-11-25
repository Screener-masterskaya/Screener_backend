package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePinger returns the configured error. The error can be changed during a test.
type fakePinger struct {
	err atomic.Pointer[error]
}

func newFakePinger(err error) *fakePinger {
	p := &fakePinger{}
	p.set(err)
	return p
}

func (p *fakePinger) set(err error) { p.err.Store(&err) }

func (p *fakePinger) Ping(context.Context) error { return *p.err.Load() }

// blockingPinger waits for context cancellation, like a hung database.
type blockingPinger struct{}

func (blockingPinger) Ping(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

var errDown = errors.New("connection refused")

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func serve(t *testing.T, handler http.HandlerFunc) (int, response) {
	t.Helper()

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	var body response
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return rec.Code, body
}

func TestLiveness(t *testing.T) {
	tests := []struct {
		name         string
		pingErr      error
		shuttingDown bool
	}{
		{name: "dependencies ok"},
		{name: "database down", pingErr: errDown},
		{name: "shutting down", shuttingDown: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(discardLogger(), time.Second, Check{Name: "postgres", Pinger: newFakePinger(tt.pingErr)})
			if tt.shuttingDown {
				h.SetShuttingDown()
			}

			code, body := serve(t, h.Liveness)

			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, "ok", body.Status)
		})
	}
}

func TestReadiness(t *testing.T) {
	tests := []struct {
		name         string
		checks       []Check
		shuttingDown bool
		wantCode     int
		wantBody     response
	}{
		{
			name:     "no checks",
			wantCode: http.StatusOK,
			wantBody: response{Status: "ready"},
		},
		{
			name:     "database up",
			checks:   []Check{{Name: "postgres", Pinger: newFakePinger(nil)}},
			wantCode: http.StatusOK,
			wantBody: response{Status: "ready"},
		},
		{
			name:     "database down",
			checks:   []Check{{Name: "postgres", Pinger: newFakePinger(errDown)}},
			wantCode: http.StatusServiceUnavailable,
			wantBody: response{Status: "not_ready", Failed: "postgres"},
		},
		{
			name:         "shutting down",
			checks:       []Check{{Name: "postgres", Pinger: newFakePinger(nil)}},
			shuttingDown: true,
			wantCode:     http.StatusServiceUnavailable,
			wantBody:     response{Status: "shutting_down"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(discardLogger(), time.Second, tt.checks...)
			if tt.shuttingDown {
				h.SetShuttingDown()
			}

			code, body := serve(t, h.Readiness)

			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func TestReadinessRecoversWithoutRestart(t *testing.T) {
	db := newFakePinger(errDown)
	h := New(discardLogger(), time.Second, Check{Name: "postgres", Pinger: db})

	code, _ := serve(t, h.Readiness)
	require.Equal(t, http.StatusServiceUnavailable, code)

	db.set(nil)

	code, body := serve(t, h.Readiness)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "ready", body.Status)
}

func TestReadinessRespectsTimeout(t *testing.T) {
	h := New(discardLogger(), 50*time.Millisecond, Check{Name: "postgres", Pinger: blockingPinger{}})

	start := time.Now()
	code, body := serve(t, h.Readiness)

	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Equal(t, "postgres", body.Failed)
	assert.Less(t, time.Since(start), time.Second)
}
