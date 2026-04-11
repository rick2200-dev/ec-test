package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the cart service.
type Config struct {
	HTTPPort           string
	RedisURL           string
	CartTTLSeconds     int
	CatalogServiceURL  string
	OrderServiceURL    string
	PubSubEmulatorHost string
	PubSubProjectID    string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPPort:           getEnv("HTTP_PORT", "8088"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		CartTTLSeconds:     getEnvInt("CART_TTL_SECONDS", 60*60*24*30), // 30 days
		CatalogServiceURL:  getEnv("CATALOG_SERVICE_URL", "http://localhost:8082"),
		OrderServiceURL:    getEnv("ORDER_SERVICE_URL", "http://localhost:8084"),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", ""),
		PubSubProjectID:    getEnv("PUBSUB_PROJECT_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
