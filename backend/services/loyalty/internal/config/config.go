package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the loyalty service.
type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string

	// Shared secret required on every non-health request. Gateway and the
	// order service must send this value via `X-Internal-Token` (HTTP) or
	// `x-internal-token` gRPC metadata. Empty causes both transports to
	// fail closed.
	InternalToken string

	// PubSubProjectID is the GCP project for publishing loyalty events
	// (PointsEarned / PointsRedeemed). Empty disables publishing, which
	// is only safe for tests or local dev without a broker.
	PubSubProjectID string

	// EarnRateBps is the points earned per currency minor unit, expressed
	// in basis points (100 = 1%). floor(paid_subtotal * EarnRateBps /
	// 10000) is inserted into the ledger on order.paid.
	EarnRateBps int

	// ReservationTTLMinutes is how long a pending point reservation
	// survives before the reaper expires it and returns the held points
	// to pending_redemption.
	ReservationTTLMinutes int
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://loyalty_role:localdev@localhost:5445/loyalty_dev?sslmode=disable"),
		HTTPPort:              getEnv("HTTP_PORT", "8094"),
		GRPCPort:              getEnv("GRPC_PORT", "50061"),
		InternalToken:         getEnv("LOYALTY_INTERNAL_TOKEN", ""),
		PubSubProjectID:       getEnv("PUBSUB_PROJECT_ID", ""),
		EarnRateBps:           getEnvInt("EARN_RATE_BPS", 100),
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
