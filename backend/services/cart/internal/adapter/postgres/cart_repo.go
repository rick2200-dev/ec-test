package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
)

// CartRepository persists carts in Redis under the key
// "cart:{tenant_id}:{buyer_auth0_id}" with a configurable TTL that
// resets on every write.
type CartRepository struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewCartRepository creates a new CartRepository. ttlSeconds controls
// how long a cart lives in Redis before expiring; the TTL is refreshed
// on every Save call, so an active cart effectively never expires.
func NewCartRepository(client *goredis.Client, ttlSeconds int) *CartRepository {
	return &CartRepository{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// Key builds the Redis key for a given (tenant, buyer) pair.
func (r *CartRepository) Key(tenantID uuid.UUID, buyerAuth0ID string) string {
	return fmt.Sprintf("cart:%s:%s", tenantID.String(), buyerAuth0ID)
}

// Get loads a cart from Redis. Returns (nil, nil) when the cart does
// not exist — callers should treat that as an empty cart.
func (r *CartRepository) Get(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.Cart, error) {
	data, err := r.client.Get(ctx, r.Key(tenantID, buyerAuth0ID)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get cart: %w", err)
	}

	var cart domain.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, fmt.Errorf("unmarshal cart: %w", err)
	}
	return &cart, nil
}

// Save serializes a cart to JSON and writes it with the configured TTL.
// Always refreshes the TTL on every write.
func (r *CartRepository) Save(ctx context.Context, cart *domain.Cart) error {
	cart.UpdatedAt = time.Now().UTC()

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("marshal cart: %w", err)
	}

	if err := r.client.Set(ctx, r.Key(cart.TenantID, cart.BuyerAuth0ID), data, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set cart: %w", err)
	}
	return nil
}

// Delete removes a cart from Redis. Used after a successful checkout.
func (r *CartRepository) Delete(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) error {
	if err := r.client.Del(ctx, r.Key(tenantID, buyerAuth0ID)).Err(); err != nil {
		return fmt.Errorf("redis del cart: %w", err)
	}
	return nil
}
