package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://screener:example@localhost:5432/screener"

func resetConfigEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"APP_ENV", "HTTP_ADDR", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT", "HTTP_SHUTDOWN_TIMEOUT", "LOG_LEVEL", "LOG_FORMAT",
		"METRICS_ADDR", "DATABASE_URL", "DB_MAX_CONNS", "DB_MAX_CONN_LIFETIME", "DB_CONNECT_TIMEOUT",
	} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DATABASE_URL", testDatabaseURL)
}

func TestLoadDefaults(t *testing.T) {
	resetConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != "development" || cfg.LogLevel != "info" ||
		cfg.LogFormat != "text" || cfg.MetricsAddr != "localhost:9090" {
		t.Fatal("unexpected application defaults")
	}
	wantHTTP := HTTPServer{"localhost:8080", 5 * time.Second, 20 * time.Second, time.Minute, 10 * time.Second}
	if cfg.HTTPServer != wantHTTP {
		t.Errorf("HTTPServer = %+v, want %+v", cfg.HTTPServer, wantHTTP)
	}
	wantPool := PostgresPool{10, time.Hour, 5 * time.Second}
	if cfg.Postgres != wantPool {
		t.Errorf("Postgres = %+v, want %+v", cfg.Postgres, wantPool)
	}
}

func TestLoadOverrides(t *testing.T) {
	resetConfigEnv(t)
	for name, value := range map[string]string{
		"APP_ENV":               "production",
		"HTTP_ADDR":             ":8000",
		"HTTP_READ_TIMEOUT":     "1s",
		"HTTP_WRITE_TIMEOUT":    "2s",
		"HTTP_IDLE_TIMEOUT":     "3s",
		"HTTP_SHUTDOWN_TIMEOUT": "4s",
		"LOG_LEVEL":             "warn",
		"LOG_FORMAT":            "json",
		"METRICS_ADDR":          ":9000",
		"DB_MAX_CONNS":          "5",
		"DB_MAX_CONN_LIFETIME":  "45m",
		"DB_CONNECT_TIMEOUT":    "2s",
	} {
		t.Setenv(name, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != "production" || cfg.LogLevel != "warn" ||
		cfg.LogFormat != "json" || cfg.MetricsAddr != ":9000" || cfg.DatabaseURL != testDatabaseURL {
		t.Fatal("application settings did not match the environment")
	}
	wantHTTP := HTTPServer{":8000", time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second}
	if cfg.HTTPServer != wantHTTP {
		t.Errorf("HTTPServer = %+v, want %+v", cfg.HTTPServer, wantHTTP)
	}
	wantPool := PostgresPool{5, 45 * time.Minute, 2 * time.Second}
	if cfg.Postgres != wantPool {
		t.Errorf("Postgres = %+v, want %+v", cfg.Postgres, wantPool)
	}
}

func TestLoadValidation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		variable string
		value    string
	}{
		{"empty database URL", "DATABASE_URL", ""},
		{"invalid database URL", "DATABASE_URL", "postgres://user:test-secret@localhost:%"},
		{"invalid database port", "DATABASE_URL", "postgres://user:test-secret@localhost:65536/db"},
		{"empty app environment", "APP_ENV", ""},
		{"invalid HTTP address", "HTTP_ADDR", "localhost"},
		{"invalid metrics port", "METRICS_ADDR", ":65536"},
		{"invalid log level", "LOG_LEVEL", "verbose"},
		{"invalid log format", "LOG_FORMAT", "xml"},
		{"invalid shutdown timeout", "HTTP_SHUTDOWN_TIMEOUT", "abc"},
		{"zero connection timeout", "DB_CONNECT_TIMEOUT", "0s"},
		{"zero pool maximum", "DB_MAX_CONNS", "0"},
		{"overflowing pool maximum", "DB_MAX_CONNS", "2147483648"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tt.variable, tt.value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.variable) {
				t.Fatalf("expected an error naming %s", tt.variable)
			}
			if strings.Contains(err.Error(), "test-secret") {
				t.Fatal("configuration error exposed a password")
			}
		})
	}
}
