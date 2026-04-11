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
	CartServiceURL         string
	InquiryServiceURL      string

	// JWT / Auth0 settings
	JWTIssuer   string
	JWTAudience string
	JWKSURL     string

	// Shared secret used to call the auth service /internal/authz/* endpoints.
	// Must match AUTH_INTERNAL_TOKEN on the auth service.
	AuthInternalToken string

	// gRPC service addresses (host:port)
	CatalogGRPCAddr   string
	InventoryGRPCAddr string
	OrderGRPCAddr     string

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
		CartServiceURL:         getEnv("CART_SERVICE_URL", "http://localhost:8088"),
		InquiryServiceURL:      getEnv("INQUIRY_SERVICE_URL", "http://localhost:8090"),

		JWTIssuer:   getEnv("JWT_ISSUER", "https://ecmarket.example.com/"),
		JWTAudience: getEnv("JWT_AUDIENCE", "https://api.ecmarket.example.com"),
		JWKSURL:     getEnv("JWKS_URL", "https://ecmarket.example.com/.well-known/jwks.json"),

		AuthInternalToken: getEnv("AUTH_INTERNAL_TOKEN", ""),

		// Note: catalog's GRPC_PORT default is 50052 (see
		// backend/services/catalog/internal/config/config.go). Keep this in
		// sync — the buyer read path now talks to catalog over gRPC.
		CatalogGRPCAddr:   getEnv("CATALOG_GRPC_ADDR", "localhost:50052"),
		OrderGRPCAddr:     getEnv("ORDER_GRPC_ADDR", "localhost:50053"),
		InventoryGRPCAddr: getEnv("INVENTORY_GRPC_ADDR", "localhost:50054"),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
