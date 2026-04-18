package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the coupon service.
type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string

	// Shared secret required on every non-health request. Gateway and
	// the order service must send this value via `X-Internal-Token`
	// (HTTP) or `x-internal-token` gRPC metadata. Empty causes both
	// transports to fail closed.
	InternalToken string

	// PubSubProjectID is the GCP project for publishing coupon events
	// (CouponIssued / CouponRedeemed / CouponRevoked). Empty disables
	// publishing — safe only for tests or local dev without a broker.
	PubSubProjectID string

	// ReservationTTLMinutes is how long a pending coupon reservation
	// survives before the reaper marks it expired and decrements
	// coupons.usage_count.
	ReservationTTLMinutes int
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://coupon_role:localdev@localhost:5444/coupon_dev?sslmode=disable"),
		HTTPPort:              getEnv("HTTP_PORT", "8093"),
		GRPCPort:              getEnv("GRPC_PORT", "50060"),
		InternalToken:         getEnv("COUPON_INTERNAL_TOKEN", ""),
		PubSubProjectID:       getEnv("PUBSUB_PROJECT_ID", ""),
		ReservationTTLMinutes: getEnvInt("RESERVATION_TTL_MINUTES", 30),
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
