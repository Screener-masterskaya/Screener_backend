// Package logger builds the service-wide structured logger on top of log/slog.
package logger

import (
	"fmt"
	"io"
	"log/slog"
)

// Supported values of the format argument of New.
const (
	FormatText = "text"
	FormatJSON = "json"
)

// New returns a logger that writes to w in the given format ("text" or "json")
// and drops records below the given level ("debug", "info", "warn" or "error").
//
// Unknown values are rejected instead of silently replaced by a default,
// so a typo in the configuration is visible at startup.
func New(w io.Writer, level, format string) (*slog.Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{Level: lvl}

	switch format {
	case FormatJSON:
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case FormatText:
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("unknown log format %q", format)
	}
}

func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", level)
	}
}
