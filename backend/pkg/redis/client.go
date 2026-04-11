// Package redis wraps github.com/redis/go-redis/v9 with a constructor
// that parses a redis:// URL, applies sensible defaults, and verifies
// connectivity before returning. Shared across services so each consumer
// does not need to re-import go-redis directly or duplicate ping plumbing.
//
// Usage:
//
//	rdb, err := pkgredis.NewClient(ctx, "redis://redis:6379/8")
//	if err != nil { ... }
//	defer rdb.Close()
package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Client is a re-exported alias over go-redis's Client type. Callers use
// pkgredis.Client so the go-redis import does not leak into every call
// site; if we ever replace the underlying library, only this package has
// to change.
type Client = goredis.Client

// NewClient creates a Redis client from a redis:// URL and pings it
// with the provided context. A failed ping closes the underlying client
// before returning so callers do not have to handle a half-open client.
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
