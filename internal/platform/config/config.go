package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	EvidenceDirectory      string
	AllowedOrigin          string
	AdminBootstrapKey      string
	TelemetryRetentionDays int
}

func Load() (Config, error) {
	config := Config{
		Port:                   env("PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		EvidenceDirectory:      env("EVIDENCE_DIRECTORY", "./storage/evidence"),
		AllowedOrigin:          env("ALLOWED_ORIGIN", "*"),
		AdminBootstrapKey:      os.Getenv("ADMIN_BOOTSTRAP_KEY"),
		TelemetryRetentionDays: IntEnv("TELEMETRY_RETENTION_DAYS", 90),
	}
	if config.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return config, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func IntEnv(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}
