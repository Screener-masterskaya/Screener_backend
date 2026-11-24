package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds top-level runtime and HTTP server settings loaded from a config file.
// TODO: support env-only overrides without a file
type Config struct {
	LogLevel   string `yaml:"log_level"`
	LogFormat  string `yaml:"log_format"`
	HTTPServer `yaml:"http_server"`
}

// HTTPServer defines network bind address and connection/shutdown timeouts.
// TODO: support env-only overrides without a file
type HTTPServer struct {
	Address         string        `yaml:"address" env-default:"localhost:8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env-default:"20s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

// MustLoad reads the configuration file specified by CONFIG_PATH and terminates the process on failure.
func MustLoad() *Config {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", err)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
