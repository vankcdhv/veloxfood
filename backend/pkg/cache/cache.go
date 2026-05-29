package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	// Scan returns all keys matching the glob pattern. Uses SCAN cursor internally.
	// Do NOT use in hot paths; intended for cache invalidation only.
	Scan(ctx context.Context, pattern string) ([]string, error)
}
