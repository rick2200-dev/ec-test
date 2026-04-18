package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the order service.
type Config struct {
	DatabaseURL         string
	HTTPPort            string
	GRPCPort            string
	StripeSecretKey     string
	StripeWebhookSecret string
	PubSubProjectID    string
	PubSubEmulatorHost string
	AuthServiceURL     string
	// SubscriptionServiceGRPCAddr is the dial target for the subscription
	// service's gRPC listener. The order service uses it to look up buyer
	// free-shipping entitlements during checkout.
	SubscriptionServiceGRPCAddr string
	// CatalogServiceGRPCAddr is the dial target for catalog's gRPC
	// listener. Used by the order service to resolve sku_id -> product_id
	// snapshots at checkout (replaces direct reads of catalog_svc).
	CatalogServiceGRPCAddr string
	// CatalogInternalToken is the shared secret attached to every RPC
	// against CatalogServiceGRPCAddr.
	CatalogInternalToken string
	// AuthInternalToken is the shared secret sent as X-Internal-Token to
	// auth's /internal/* HTTP endpoints (specifically /internal/sellers/
	// batch-get at checkout).
	AuthInternalToken  string
	DefaultShippingFee int64

	// Shared secret required on every request to /internal/*. Must match
	// the value set on in-cluster callers (cart, inquiry). Empty value
	// causes the middleware to fail closed with 503.
	InternalToken string

	// Outbound shared secret presented to the subscription service on the
	// gRPC free-shipping lookup. Must match SUBSCRIPTION_INTERNAL_TOKEN
	// on the subscription service; without it every HasFreeShipping call
	// will fail Unauthenticated.
	SubscriptionInternalToken string

	// Coupon + loyalty integration. URLs point at the coupon and
	// loyalty HTTP services; tokens gate the /internal/* routes. The
	// Enable* flags decide whether CreateCheckout / HandlePaymentSuccess
	// actually call those services — we default to off so a fresh
	// deployment stays safe until the flag is flipped.
	CouponServiceURL     string
	CouponInternalToken  string
	LoyaltyServiceURL    string
	LoyaltyInternalToken string
	EnableCoupons        bool
	EnableLoyalty        bool
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://ecmarket:ecmarket@localhost:5432/ecmarket?sslmode=disable"),
		HTTPPort:            getEnv("HTTP_PORT", "8083"),
		GRPCPort:            getEnv("GRPC_PORT", "50053"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PubSubProjectID:    getEnv("PUBSUB_PROJECT_ID", ""),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", ""),
		AuthServiceURL:              getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		SubscriptionServiceGRPCAddr: getEnv("SUBSCRIPTION_SERVICE_GRPC_ADDR", "localhost:50058"),
		CatalogServiceGRPCAddr:      getEnv("CATALOG_SERVICE_GRPC_ADDR", "localhost:50054"),
		CatalogInternalToken:        getEnv("CATALOG_INTERNAL_TOKEN", ""),
		AuthInternalToken:           getEnv("AUTH_INTERNAL_TOKEN", ""),
		DefaultShippingFee:          getEnvInt64("DEFAULT_SHIPPING_FEE", 500),
		InternalToken:             getEnv("ORDER_INTERNAL_TOKEN", ""),
		SubscriptionInternalToken: getEnv("SUBSCRIPTION_INTERNAL_TOKEN", ""),

		CouponServiceURL:     getEnv("COUPON_SERVICE_URL", "http://localhost:8093"),
		CouponInternalToken:  getEnv("COUPON_INTERNAL_TOKEN", ""),
		LoyaltyServiceURL:    getEnv("LOYALTY_SERVICE_URL", "http://localhost:8094"),
		LoyaltyInternalToken: getEnv("LOYALTY_INTERNAL_TOKEN", ""),
		EnableCoupons:        getEnvBool("ENABLE_COUPONS", false),
		EnableLoyalty:        getEnvBool("ENABLE_LOYALTY", false),
	}
}

// getEnvBool accepts "true" / "1" / "yes" (case-insensitive); everything
// else — including unset — is false. Keeps feature flags explicit.
func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "true", "TRUE", "True", "1", "yes", "YES", "Yes":
		return true
	}
	return false
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
