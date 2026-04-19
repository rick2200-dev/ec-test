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

	// PointExpirationDays is the runway stamped on each new earn row's
	// expires_at. The reaper retires earns whose expires_at has passed.
	// Zero or negative disables the feature (earn rows stay permanent).
	// Must match the INTERVAL used in migration 003's backfill SQL.
	PointExpirationDays int

	// PointExpirationIntervalMinutes is how often the expiration
	// reaper fires. Defaults to 60 — expiration is a slow-timescale
	// operation, unlike the 60s reservation reaper.
	PointExpirationIntervalMinutes int

	// PointExpirationBatchSize caps buyers processed per expiration
	// tick so a backlog can't stall the whole service on startup.
	PointExpirationBatchSize int

	// PointExpirationEnabled is the kill switch — default off during
	// rollout so operators can flip it on AFTER backfill verification.
	PointExpirationEnabled bool

	// PointExpirationTickTimeoutSeconds bounds how long a single
	// reaper tick can run. Full-ledger walks for many buyers can be
	// slow; default 120s is generous.
	PointExpirationTickTimeoutSeconds int
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:                       getEnv("DATABASE_URL", "postgres://loyalty_role:localdev@localhost:5445/loyalty_dev?sslmode=disable"),
		HTTPPort:                          getEnv("HTTP_PORT", "8094"),
		GRPCPort:                          getEnv("GRPC_PORT", "50061"),
		InternalToken:                     getEnv("LOYALTY_INTERNAL_TOKEN", ""),
		PubSubProjectID:                   getEnv("PUBSUB_PROJECT_ID", ""),
		EarnRateBps:                       getEnvInt("EARN_RATE_BPS", 100),
		ReservationTTLMinutes:             getEnvInt("RESERVATION_TTL_MINUTES", 30),
		PointExpirationDays:               getEnvInt("POINT_EXPIRATION_DAYS", 365),
		PointExpirationIntervalMinutes:    getEnvInt("POINT_EXPIRATION_INTERVAL_MINUTES", 60),
		PointExpirationBatchSize:          getEnvInt("POINT_EXPIRATION_BATCH_SIZE", 200),
		PointExpirationEnabled:            getEnvBool("POINT_EXPIRATION_ENABLED", false),
		PointExpirationTickTimeoutSeconds: getEnvInt("POINT_EXPIRATION_TICK_TIMEOUT_SECONDS", 120),
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

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
