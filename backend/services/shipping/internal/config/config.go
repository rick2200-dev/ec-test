package config

import "os"

// Config holds all configuration for the shipping service.
type Config struct {
	HTTPPort string
	GRPCPort string

	DatabaseURL string

	PubSubProjectID    string
	PubSubEmulatorHost string

	InternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPPort: getEnv("HTTP_PORT", "8092"),
		GRPCPort: getEnv("GRPC_PORT", "50059"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ecmarket?sslmode=disable"),

		PubSubProjectID:    getEnv("PUBSUB_PROJECT_ID", ""),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", ""),

		InternalToken: getEnv("SHIPPING_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
