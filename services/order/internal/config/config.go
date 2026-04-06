package config

import (
	"os"
)

// Config holds all configuration for the order service.
type Config struct {
	DatabaseURL        string
	HTTPPort           string
	StripeSecretKey    string
	StripeWebhookSecret string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:           getEnv("HTTP_PORT", "8083"),
		StripeSecretKey:    getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
