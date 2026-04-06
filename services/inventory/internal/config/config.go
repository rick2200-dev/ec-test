package config

import (
	"os"
)

// Config holds all configuration for the inventory service.
type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:    getEnv("HTTP_PORT", "8084"),
		GRPCPort:    getEnv("GRPC_PORT", "50054"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
