package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryShowRepository(t *testing.T) {
	for _, actors := range []int{1, 4} {
		t.Run(fmt.Sprintf("actors=%d", actors), func(t *testing.T) {
			runShowRepositoryContract(t, func(t *testing.T) ShowRepository {
				r := NewMemoryShowRepository(actors)
				t.Cleanup(r.Close)
				return r
			})
		})
	}
}

func TestMemoryEpisodeRepository(t *testing.T) {
	for _, actors := range []int{1, 4} {
		t.Run(fmt.Sprintf("actors=%d", actors), func(t *testing.T) {
			runEpisodeRepositoryContract(t, func(t *testing.T) EpisodeRepository {
				r := NewMemoryEpisodeRepository(actors)
				t.Cleanup(r.Close)
				return r
			}, func(t *testing.T, showID uuid.UUID) {}) // memory: no FK, no-op

		})
	}
}

// TestSpawnsConfiguredActors proves the constructor spawns exactly one live
// goroutine per configured actor: each shard answers a ping within the deadline.
func TestSpawnsConfiguredActors(t *testing.T) {
	const n = 8
	r := NewMemoryShowRepository(n)
	t.Cleanup(r.Close)

	if r.Actors() != n {
		t.Fatalf("Actors() = %d, want %d", r.Actors(), n)
	}
	for i, sh := range r.shards {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := submit(ctx, sh.actor, func() (struct{}, error) { return struct{}{}, nil })
		cancel()
		if err != nil {
			t.Fatalf("shard %d goroutine not responding: %v", i, err)
		}
	}
}

func TestMemoryUserRepository(t *testing.T) {
	runUserRepositoryContract(t, func(t *testing.T) UserRepository {
		r := NewMemoryUserRepository()
		t.Cleanup(r.Close)
		return r
	})
}
