package config

import "os"

// Config holds all configuration for the review service.
type Config struct {
	DatabaseURL    string
	HTTPPort       string
	PubSubProjectID string

	// CatalogServiceURL is the base URL of the catalog service for
	// product lookups via /internal/products/{id}.
	CatalogServiceURL string

	// OrderServiceURL is the base URL of the order service for
	// purchase verification via /internal/purchase-check.
	OrderServiceURL string

	// CatalogInternalToken is the shared secret for X-Internal-Token
	// on requests to the catalog service /internal endpoints.
	CatalogInternalToken string

	// OrderInternalToken is the shared secret for X-Internal-Token
	// on requests to the order service /internal endpoints.
	OrderInternalToken string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:             getEnv("HTTP_PORT", "8091"),
		PubSubProjectID:      getEnv("PUBSUB_PROJECT_ID", ""),
		CatalogServiceURL:    getEnv("CATALOG_SERVICE_URL", "http://localhost:8082"),
		OrderServiceURL:      getEnv("ORDER_SERVICE_URL", "http://localhost:8083"),
		CatalogInternalToken: getEnv("CATALOG_INTERNAL_TOKEN", ""),
		OrderInternalToken:   getEnv("ORDER_INTERNAL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
