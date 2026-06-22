package cache

import "github.com/yazeedalorainy/thmanyah/internal/config"

// New builds the read-path cache for the config: Redis when enabled, otherwise
// a noop (caching off — every read falls through to the store).
func New(cfg config.CacheConfig) (Cache, func()) {
	if cfg.Redis.Enabled {
		rc := NewRedis(cfg.Redis.Addr)
		return rc, func() { _ = rc.Close() }
	}
	return NoopCache{}, func() {}
}
