package config

import (
	"os"
	"strings"
)

// Config holds all configuration for the search service.
type Config struct {
	DatabaseURL       string
	HTTPPort          string
	VertexAIEnabled   bool
	GCPProjectID      string
	PubSubProjectID   string
	AuthServiceURL    string
	AuthInternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:          getEnv("HTTP_PORT", "8085"),
		VertexAIEnabled:   strings.EqualFold(getEnv("VERTEX_AI_ENABLED", "false"), "true"),
		GCPProjectID:      getEnv("GCP_PROJECT_ID", ""),
		PubSubProjectID:   getEnv("PUBSUB_PROJECT_ID", ""),
		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://auth:8081"),
		AuthInternalToken: getEnv("AUTH_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
