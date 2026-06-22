package store_test

import (
	"testing"

	"github.com/yazeedalorainy/thmanyah/internal/config"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

// TestActorsFromConfig shows the wiring main.go will use: the configured
// memory_actors value drives how many shard goroutines the repository spawns.
func TestActorsFromConfig(t *testing.T) {
	cfg, err := config.Load("../../config.example.yaml")
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	repo := store.NewMemoryShowRepository(cfg.Store.Memory.Actors)
	t.Cleanup(repo.Close)

	if repo.Actors() != cfg.Store.Memory.Actors {
		t.Fatalf("Actors() = %d, want %d (from config)", repo.Actors(), cfg.Store.Memory.Actors)
	}
}
