package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache is a byte-blob key/value store with TTL. Adapters: Redis, in-memory, noop.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error) // bool reports a hit
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// group coalesces concurrent misses for the same key (stampede protection).
var group singleflight.Group

// Remember returns the cached value for key, or computes it via load, stores it
// with ttl, and returns it. Concurrent misses for the same key run load once.
// Load errors are returned and not cached.
func Remember[T any](ctx context.Context, c Cache, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	if b, ok, err := c.Get(ctx, key); err == nil && ok {
		var v T
		if err := json.Unmarshal(b, &v); err == nil {
			return v, nil
		} else {
			slog.WarnContext(ctx, "cache value decode failed", "key", key, "error", err)
		}
	} else if err != nil {
		slog.WarnContext(ctx, "cache get failed", "key", key, "error", err)
	}
	res, err, _ := group.Do(key, func() (any, error) {
		v, err := load(ctx)
		if err != nil {
			return nil, err
		}
		if b, mErr := json.Marshal(v); mErr == nil {
			if err := c.Set(ctx, key, b, ttl); err != nil {
				slog.WarnContext(ctx, "cache set failed", "key", key, "ttl", ttl, "error", err)
			}
		} else {
			slog.WarnContext(ctx, "cache value encode failed", "key", key, "error", mErr)
		}
		return v, nil
	})
	if err != nil {
		return zero, err
	}
	return res.(T), nil
}

// NoopCache disables caching: every Get misses, Set/Delete do nothing.
type NoopCache struct{}

func (NoopCache) Get(context.Context, string) ([]byte, bool, error)        { return nil, false, nil }
func (NoopCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (NoopCache) Delete(context.Context, string) error                     { return nil }
