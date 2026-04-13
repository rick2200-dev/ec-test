package config

import "os"

// Config holds all configuration for the subscription service.
type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string

	// Shared secret required on every non-health request. Gateway and the
	// order service must send the same value via `X-Internal-Token`
	// (HTTP) or `x-internal-token` gRPC metadata. Empty value causes the
	// HTTP middleware to fail closed with 503 and the gRPC interceptor
	// to reject with Unauthenticated.
	InternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:      getEnv("HTTP_PORT", "8089"),
		GRPCPort:      getEnv("GRPC_PORT", "50058"),
		InternalToken: getEnv("SUBSCRIPTION_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
