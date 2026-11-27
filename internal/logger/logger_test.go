package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Screener-masterskaya/Screener_backend/internal/logger"
)

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	log, err := logger.New(&buf, "info", logger.FormatJSON)
	require.NoError(t, err)

	log.Info("hello", "key", "value")

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	require.Equal(t, "INFO", rec["level"])
	require.Equal(t, "hello", rec["msg"])
	require.Equal(t, "value", rec["key"])
}

func TestNew_LevelFiltering(t *testing.T) {
	// Error records pass every supported level, so they need no table field.
	tests := []struct {
		level     string
		wantDebug bool
		wantInfo  bool
		wantWarn  bool
	}{
		{level: "debug", wantDebug: true, wantInfo: true, wantWarn: true},
		{level: "info", wantInfo: true, wantWarn: true},
		{level: "warn", wantWarn: true},
		{level: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			var buf bytes.Buffer

			log, err := logger.New(&buf, tt.level, logger.FormatText)
			require.NoError(t, err)

			log.Debug("debug message")
			log.Info("info message")
			log.Warn("warn message")
			log.Error("error message")

			out := buf.String()
			require.Equal(t, tt.wantDebug, bytes.Contains(buf.Bytes(), []byte("debug message")), out)
			require.Equal(t, tt.wantInfo, bytes.Contains(buf.Bytes(), []byte("info message")), out)
			require.Equal(t, tt.wantWarn, bytes.Contains(buf.Bytes(), []byte("warn message")), out)
			require.Contains(t, out, "error message")
		})
	}
}

func TestNew_InvalidArguments(t *testing.T) {
	// Each case has exactly one invalid argument, the one named in the case.
	tests := []struct {
		name    string
		level   string
		format  string
		wantErr string
	}{
		{name: "unknown level", level: "verbose", format: logger.FormatJSON, wantErr: "unknown log level"},
		{name: "empty level", level: "", format: logger.FormatJSON, wantErr: "unknown log level"},
		{name: "unknown format", level: "info", format: "xml", wantErr: "unknown log format"},
		{name: "empty format", level: "info", format: "", wantErr: "unknown log format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := logger.New(&bytes.Buffer{}, tt.level, tt.format)

			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, log)
		})
	}
}
