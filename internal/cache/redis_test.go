//go:build integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisCache(t *testing.T) {
	ctx := context.Background()
	container, err := tcredis.Run(ctx, "redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("6379/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("endpoint: %v", err)
	}

	c := NewRedis(endpoint)
	t.Cleanup(func() { _ = c.Close() })
	runCacheContract(t, c)
}
