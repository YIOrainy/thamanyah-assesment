package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func runExternalRefContract(t *testing.T, newRepo func(t *testing.T) ExternalRefRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("create then find owner", func(t *testing.T) {
		r := newRepo(t)
		owner := uuid.Must(uuid.NewV7())
		ref := &ExternalRef{
			ID: uuid.Must(uuid.NewV7()), Source: "rss", OwnerType: OwnerEpisode,
			OwnerID: owner, ExternalID: "guid-1", ImportedAt: time.Now().UTC(),
		}
		if err := r.Create(ctx, ref); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, found, err := r.FindOwner(ctx, "rss", OwnerEpisode, "guid-1")
		if err != nil || !found || got != owner {
			t.Fatalf("FindOwner: got=%v found=%v err=%v", got, found, err)
		}
	})

	t.Run("missing ref → not found", func(t *testing.T) {
		r := newRepo(t)
		if _, found, err := r.FindOwner(ctx, "rss", OwnerEpisode, "nope"); err != nil || found {
			t.Errorf("found=%v err=%v, want not found", found, err)
		}
	})

	t.Run("duplicate key → ErrConflict", func(t *testing.T) {
		r := newRepo(t)
		mk := func() *ExternalRef {
			return &ExternalRef{ID: uuid.Must(uuid.NewV7()), Source: "rss", OwnerType: OwnerShow, OwnerID: uuid.Must(uuid.NewV7()), ExternalID: "dup"}
		}
		if err := r.Create(ctx, mk()); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := r.Create(ctx, mk()); !errors.Is(err, ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})
}

func TestMemoryExternalRefRepository(t *testing.T) {
	runExternalRefContract(t, func(t *testing.T) ExternalRefRepository {
		r := NewMemoryExternalRefRepository()
		t.Cleanup(r.Close)
		return r
	})
}
