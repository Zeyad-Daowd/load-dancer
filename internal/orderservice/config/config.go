package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	// Port               string
	ShutdownTimeout    time.Duration
	ControlPlaneSecret string
	LoadBalancerURL    string
}

func Load() (Config, error) {
	err := godotenv.Load()

	if err != nil {
		slog.Warn("could not load .env", "error", err)
	}

	cfg := Config{
		ControlPlaneSecret: os.Getenv("CONTROL_PLANE_SECRET"),
		// Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:     os.Getenv("GOOSE_DBSTRING"),
		LoadBalancerURL: envOrDefault("LOAD_BALANCER_URL", "http://localhost:8081"),
		ShutdownTimeout: 10 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("GOOSE_DBSTRING is not set")
	}

	if cfg.ControlPlaneSecret == "" {
		return cfg, fmt.Errorf("CONTROL_PLANE_SECRET is not set")
	}
	return cfg, nil
}

func envOrDefault(key string, defaultValue string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return defaultValue
}
