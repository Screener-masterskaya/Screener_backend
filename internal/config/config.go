package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv      string
	LogLevel    string
	LogFormat   string
	MetricsAddr string
	DatabaseURL string
	HTTPServer
	Postgres PostgresPool
}

type HTTPServer struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type PostgresPool struct {
	MaxConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:      envOrDefault("APP_ENV", "development"),
		LogLevel:    envOrDefault("LOG_LEVEL", "info"),
		LogFormat:   envOrDefault("LOG_FORMAT", "text"),
		MetricsAddr: envOrDefault("METRICS_ADDR", "localhost:9090"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		HTTPServer:  HTTPServer{Address: envOrDefault("HTTP_ADDR", "localhost:8080")},
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	dsn, err := url.Parse(cfg.DatabaseURL)
	if err != nil || dsn.Hostname() == "" ||
		(dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") {
		return nil, errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	}
	if port := dsn.Port(); port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 {
			return nil, errors.New("DATABASE_URL port must be between 1 and 65535")
		}
	}

	if cfg.AppEnv == "" {
		return nil, errors.New("APP_ENV must not be empty")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, errors.New("LOG_LEVEL must be debug, info, warn, or error")
	}
	switch cfg.LogFormat {
	case "text", "json":
	default:
		return nil, errors.New("LOG_FORMAT must be text or json")
	}

	for _, setting := range []struct {
		name  string
		value string
	}{
		{"HTTP_ADDR", cfg.Address},
		{"METRICS_ADDR", cfg.MetricsAddr},
	} {
		_, port, err := net.SplitHostPort(setting.value)
		if err != nil || port == "" {
			return nil, fmt.Errorf("%s must be in host:port format", setting.name)
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, fmt.Errorf("%s port must be between 0 and 65535", setting.name)
		}
	}

	for _, setting := range []struct {
		name     string
		fallback string
		dest     *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", "5s", &cfg.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", "20s", &cfg.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", "60s", &cfg.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", "10s", &cfg.ShutdownTimeout},
		{"DB_MAX_CONN_LIFETIME", "1h", &cfg.Postgres.MaxConnLifetime},
		{"DB_CONNECT_TIMEOUT", "5s", &cfg.Postgres.ConnectTimeout},
	} {
		value, err := time.ParseDuration(envOrDefault(setting.name, setting.fallback))
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("%s must be a positive duration, e.g. 5s", setting.name)
		}
		*setting.dest = value
	}
	maxConns, err := strconv.ParseInt(envOrDefault("DB_MAX_CONNS", "10"), 10, 32)
	if err != nil || maxConns < 1 {
		return nil, errors.New("DB_MAX_CONNS must be a positive 32-bit integer")
	}
	cfg.Postgres.MaxConns = int32(maxConns)

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		return fallback
	}
	return strings.TrimSpace(value)
}
