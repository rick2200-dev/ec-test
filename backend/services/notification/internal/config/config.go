package config

import (
	"os"
)

// Config holds all configuration for the notification service.
type Config struct {
	HTTPPort           string
	PubSubEmulatorHost string
	PubSubProjectID    string
	SMTPHost           string
	SMTPPort           string
	SMTPFrom           string
	// DatabaseURL is optional. When set, the service uses a Postgres-backed
	// event deduplicator to prevent duplicate notifications on at-least-once
	// re-deliveries. When unset, deduplication is skipped (no-op).
	DatabaseURL string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPPort:           getEnv("HTTP_PORT", "8087"),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", "localhost:8085"),
		PubSubProjectID:    getEnv("PUBSUB_PROJECT_ID", "ec-test-local"),
		SMTPHost:           getEnv("SMTP_HOST", "localhost"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPFrom:           getEnv("SMTP_FROM", "noreply@ecmarket.local"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
