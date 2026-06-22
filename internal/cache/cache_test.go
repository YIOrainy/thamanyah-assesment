package cache

import (
	"context"
	"testing"
	"time"
)

func runCacheContract(t *testing.T, c Cache) {
	t.Helper()
	ctx := context.Background()

	t.Run("miss then set then hit", func(t *testing.T) {
		if _, ok, _ := c.Get(ctx, "k1"); ok {
			t.Fatal("expected miss on unset key")
		}
		if err := c.Set(ctx, "k1", []byte("v1"), time.Minute); err != nil {
			t.Fatalf("Set: %v", err)
		}
		got, ok, err := c.Get(ctx, "k1")
		if err != nil || !ok {
			t.Fatalf("Get: ok=%v err=%v", ok, err)
		}
		if string(got) != "v1" {
			t.Errorf("got %q, want v1", got)
		}
	})

	t.Run("delete removes", func(t *testing.T) {
		_ = c.Set(ctx, "k2", []byte("v2"), time.Minute)
		if err := c.Delete(ctx, "k2"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, ok, _ := c.Get(ctx, "k2"); ok {
			t.Error("key still present after delete")
		}
	})
}

func TestMemoryCache(t *testing.T) {
	runCacheContract(t, NewMemory())
}

func TestRemember(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()
	calls := 0
	load := func(context.Context) (int, error) {
		calls++
		return 42, nil
	}

	v, err := Remember(ctx, c, "answer", time.Minute, load)
	if err != nil || v != 42 {
		t.Fatalf("first Remember: v=%d err=%v", v, err)
	}
	v2, _ := Remember(ctx, c, "answer", time.Minute, load) // should hit cache
	if v2 != 42 {
		t.Errorf("second Remember v=%d, want 42", v2)
	}
	if calls != 1 {
		t.Errorf("load called %d times, want 1 (second should hit cache)", calls)
	}
}
