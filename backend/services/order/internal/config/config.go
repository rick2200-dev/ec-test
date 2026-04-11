package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the order service.
type Config struct {
	DatabaseURL         string
	HTTPPort            string
	GRPCPort            string
	StripeSecretKey     string
	StripeWebhookSecret string
	PubSubProjectID     string
	AuthServiceURL      string
	DefaultShippingFee  int64

	// Shared secret required on every request to /internal/*. Must match
	// the value set on in-cluster callers (cart, inquiry). Empty value
	// causes the middleware to fail closed with 503.
	InternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:            getEnv("HTTP_PORT", "8083"),
		GRPCPort:            getEnv("GRPC_PORT", "50053"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PubSubProjectID:     getEnv("PUBSUB_PROJECT_ID", ""),
		AuthServiceURL:      getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		DefaultShippingFee:  getEnvInt64("DEFAULT_SHIPPING_FEE", 500),
		InternalToken:       getEnv("ORDER_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
