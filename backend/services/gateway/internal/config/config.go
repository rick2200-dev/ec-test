package config

import (
	"os"
	"strconv"
	"time"
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
	ReviewServiceURL       string

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

	// Redis — used by the API-token rate limiter. The cart service uses
	// DB 7; we take DB 8 so the two subsystems cannot collide on keys.
	RedisURL string

	// Seller API token settings. Prefix must match API_TOKEN_PREFIX on the
	// auth service (checked at runtime — tokens minted against a different
	// prefix will simply fail to parse here).
	APITokenPrefix       string
	APITokenCacheTTL     time.Duration
	APITokenRPSDefault   int
	APITokenBurstDefault int
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
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8087"),
		RecommendServiceURL:    getEnv("RECOMMEND_SERVICE_URL", "http://localhost:8086"),
		CartServiceURL:         getEnv("CART_SERVICE_URL", "http://localhost:8088"),
		InquiryServiceURL:      getEnv("INQUIRY_SERVICE_URL", "http://localhost:8090"),
		ReviewServiceURL:       getEnv("REVIEW_SERVICE_URL", "http://localhost:8091"),

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

		RedisURL: getEnv("REDIS_URL", "redis://redis:6379/8"),

		APITokenPrefix:       getEnv("API_TOKEN_PREFIX", "sk_live_"),
		APITokenCacheTTL:     getEnvDuration("API_TOKEN_CACHE_TTL_SECONDS", 30*time.Second),
		APITokenRPSDefault:   getEnvInt("API_TOKEN_RATE_LIMIT_DEFAULT_RPS", 10),
		APITokenBurstDefault: getEnvInt("API_TOKEN_RATE_LIMIT_DEFAULT_BURST", 20),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt reads an int from the environment, falling back on the given
// default on parse failure. Non-positive values are treated as "unset" —
// zero-RPS rate limiting would silently break API-token traffic.
func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// getEnvDuration reads a duration, in seconds, from the environment. We
// intentionally do not accept Go's native duration syntax here — the
// variable name ends in "_SECONDS" so operators see unambiguous units in
// docker-compose and secrets files.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}
