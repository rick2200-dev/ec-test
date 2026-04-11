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

	// InternalToken is the shared secret required on /internal/authz/*
	// requests from the gateway. Empty disables the internal endpoints.
	InternalToken string

	// APITokenPrefix is the wire-format prefix applied to newly issued
	// seller API access tokens (e.g. "sk_live_"). Rotatable via env so
	// operators can change the label without a code release; existing
	// tokens continue to work because each row stores its own prefix.
	APITokenPrefix string

	// BootstrapSuperAdminSub is the Auth0 "sub" of the initial super_admin
	// to seed on startup if none exists for BootstrapTenantID. Both must be
	// set for the bootstrap to run.
	BootstrapSuperAdminSub string
	BootstrapTenantID      string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		HTTPPort:               getEnv("HTTP_PORT", "8081"),
		GRPCPort:               getEnv("GRPC_PORT", "50051"),
		InternalToken:          getEnv("AUTH_INTERNAL_TOKEN", ""),
		APITokenPrefix:         getEnv("API_TOKEN_PREFIX", "sk_live_"),
		BootstrapSuperAdminSub: getEnv("AUTH_BOOTSTRAP_SUPERADMIN_SUB", ""),
		BootstrapTenantID:      getEnv("AUTH_BOOTSTRAP_TENANT_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
