package config

import (
	"os"
)

// Config holds all configuration for the catalog service.
type Config struct {
	DatabaseURL        string
	HTTPPort           string
	GRPCPort           string
	PubSubEmulatorHost string
	PubSubProjectID    string

	// Shared secret required on every request to /internal/*. Must match the
	// value set on any in-cluster caller (cart service today). Empty value
	// causes the middleware to fail closed with 503.
	InternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:           getEnv("HTTP_PORT", "8082"),
		GRPCPort:           getEnv("GRPC_PORT", "50052"),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", ""),
		PubSubProjectID:    getEnv("PUBSUB_PROJECT_ID", ""),
		InternalToken:      getEnv("CATALOG_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
