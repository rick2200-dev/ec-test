package config

import "os"

// Config holds all configuration for the inquiry service.
type Config struct {
	DatabaseURL       string
	HTTPPort          string
	OrderServiceURL   string
	AuthServiceURL    string
	InternalAuthToken string
	PubSubProjectID   string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:          getEnv("HTTP_PORT", "8090"),
		OrderServiceURL:   getEnv("ORDER_SERVICE_URL", "http://localhost:8083"),
		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		InternalAuthToken: getEnv("INTERNAL_AUTH_TOKEN", ""),
		PubSubProjectID:   getEnv("PUBSUB_PROJECT_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
