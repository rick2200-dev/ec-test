package config

import (
	"os"
)

// Config holds all configuration for the gateway service.
type Config struct {
	HTTPPort string

	// Downstream service URLs
	AuthServiceURL         string
	CatalogServiceURL      string
	OrderServiceURL        string
	InventoryServiceURL    string
	SearchServiceURL       string
	NotificationServiceURL string
	RecommendServiceURL    string

	// JWT / Auth0 settings
	JWTIssuer   string
	JWTAudience string
	JWKSURL     string

	// CORS
	AllowedOrigins string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),

		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		CatalogServiceURL:      getEnv("CATALOG_SERVICE_URL", "http://localhost:8082"),
		OrderServiceURL:        getEnv("ORDER_SERVICE_URL", "http://localhost:8083"),
		InventoryServiceURL:    getEnv("INVENTORY_SERVICE_URL", "http://localhost:8084"),
		SearchServiceURL:       getEnv("SEARCH_SERVICE_URL", "http://localhost:8085"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8086"),
		RecommendServiceURL:    getEnv("RECOMMEND_SERVICE_URL", "http://localhost:8087"),

		JWTIssuer:   getEnv("JWT_ISSUER", "https://ecmarket.example.com/"),
		JWTAudience: getEnv("JWT_AUDIENCE", "https://api.ecmarket.example.com"),
		JWKSURL:     getEnv("JWKS_URL", "https://ecmarket.example.com/.well-known/jwks.json"),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
