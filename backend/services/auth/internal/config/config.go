package config

import (
	"os"
)

// Config holds all configuration for the auth service.
type Config struct {
	DatabaseURL string
	RedisURL    string
	HTTPPort    string
	GRPCPort    string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		HTTPPort:    getEnv("HTTP_PORT", "8081"),
		GRPCPort:    getEnv("GRPC_PORT", "50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
