// Package redis wraps the github.com/redis/go-redis/v9 client with a
// constructor that parses a redis:// URL, applies sensible defaults, and
// verifies connectivity before returning. The cart service is currently
// the only Redis consumer in the monorepo; if additional services adopt
// Redis this helper should be promoted to backend/pkg/redis.
package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Client is a thin alias over go-redis's Client type, exposed so callers
// do not have to import go-redis directly.
type Client = goredis.Client

// NewClient creates a Redis client from a redis:// URL and pings it.
func NewClient(ctx context.Context, url string) (*Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := goredis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
