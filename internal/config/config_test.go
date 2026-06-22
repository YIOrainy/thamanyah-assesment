package config

import (
	"testing"
	"time"
)

func TestLoadExample(t *testing.T) {
	c, err := Load("../../config.example.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Store.Memory.Actors != 4 {
		t.Errorf("Actors = %d, want 4", c.Store.Memory.Actors)
	}
	if c.Server.CMSPort != "8080" {
		t.Errorf("CMSPort = %q, want 8080", c.Server.CMSPort)
	}
	if !c.Store.Postgres.Enabled {
		t.Error("Store.Postgres.Enabled = false, want true")
	}
	if c.Cache.Redis.TTL.Duration() != time.Minute {
		t.Errorf("Redis.TTL = %s, want 1m", c.Cache.Redis.TTL.Duration())
	}
}
