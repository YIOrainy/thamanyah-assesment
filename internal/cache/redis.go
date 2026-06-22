package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a Redis-backed Cache, shared across service instances.
type RedisCache struct {
	client *redis.Client
}

func NewRedis(addr string) *RedisCache {
	return &RedisCache{client: redis.NewClient(&redis.Options{Addr: addr})}
}

var _ Cache = (*RedisCache)(nil)

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Close() error { return c.client.Close() }
