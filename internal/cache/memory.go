package cache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	value []byte
	exp   time.Time // zero = no expiry
}

// MemoryCache is an in-process TTL cache, used for tests and the cache contract
// suite. (Real deployments use Redis so the cache is shared across instances.)
type MemoryCache struct {
	mu    sync.Mutex
	items map[string]entry
}

func NewMemory() *MemoryCache {
	return &MemoryCache{items: make(map[string]entry)}
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return nil, false, nil
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		delete(c.items, key)
		return nil, false, nil
	}
	return e.value, true, nil
}

func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.items[key] = entry{value: value, exp: exp}
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}
