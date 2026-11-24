package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Screener-masterskaya/Screener_backend/internal/app"
	"github.com/Screener-masterskaya/Screener_backend/internal/config"
)

const (
	lvlDebug = "debug"
	lvlInfo  = "info"
	lvlWarn  = "warn"
	lvlError = "error"
	logJSON  = "json"
	logText  = "text"
)

func main() {

	//TODO : refactor this to env
	//init config
	cfg := config.MustLoad()
	//init logger
	log := setupLogger(cfg.LogLevel, cfg.LogFormat)

	app := app.NewApp(cfg, log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Error("stopped with error", Err(err))
		os.Exit(1)
	}
}

// setupLogger configures and returns a structured logger for the given level and format.
// It falls back to a text debug logger when an unrecognized format is provided.
// TODO: move to a separate package with a logger
func setupLogger(logLevel, logFormat string) *slog.Logger {
	var log *slog.Logger

	switch logFormat {
	case logJSON:
		switch logLevel {
		case lvlDebug:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		case lvlInfo:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		case lvlWarn:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		case lvlError:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		default:
			log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	case logText:
		switch logLevel {
		case lvlDebug:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		case lvlInfo:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		case lvlWarn:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
		case lvlError:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		default:
			log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	default:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}

// Err formats an error as a structured slog attribute under the "error" key.
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
