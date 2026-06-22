package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

type showRepoFactory func(t *testing.T) ShowRepository

func runShowRepositoryContract(t *testing.T, newRepo showRepoFactory) {
	t.Helper()
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV7())

	t.Run("Create then GetByID round-trips", func(t *testing.T) {
		r := newRepo(t)
		show := catalog.NewShow("فنجان", "finjan", "بودكاست", catalog.FormatPodcast, "ar", owner)
		if err := r.Create(ctx, show); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := r.GetByID(ctx, show.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Title != "فنجان" || got.Slug != "finjan" {
			t.Errorf("round-trip mismatch: %+v", got)
		}
	})

	t.Run("GetByID missing returns ErrNotFound", func(t *testing.T) {
		r := newRepo(t)
		if _, err := r.GetByID(ctx, uuid.Must(uuid.NewV7())); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("duplicate slug returns ErrConflict", func(t *testing.T) {
		r := newRepo(t)
		if err := r.Create(ctx, catalog.NewShow("A", "dup", "", catalog.FormatPodcast, "ar", owner)); err != nil {
			t.Fatalf("Create A: %v", err)
		}
		if err := r.Create(ctx, catalog.NewShow("B", "dup", "", catalog.FormatPodcast, "ar", owner)); !errors.Is(err, ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("Update persists changes", func(t *testing.T) {
		r := newRepo(t)
		show := catalog.NewShow("old", "slug-x", "", catalog.FormatPodcast, "ar", owner)
		if err := r.Create(ctx, show); err != nil {
			t.Fatalf("Create: %v", err)
		}
		show.Title = "new"
		show.Status = catalog.StatusPublished
		if err := r.Update(ctx, show); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, _ := r.GetByID(ctx, show.ID)
		if got.Title != "new" || got.Status != catalog.StatusPublished {
			t.Errorf("update not persisted: %+v", got)
		}
	})

	t.Run("Update missing returns ErrNotFound", func(t *testing.T) {
		r := newRepo(t)
		show := catalog.NewShow("x", "y", "", catalog.FormatPodcast, "ar", owner)
		if err := r.Update(ctx, show); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("List on empty returns non-nil slice", func(t *testing.T) {
		r := newRepo(t)
		got, err := r.List(ctx, ShowFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("List returned nil, want non-nil empty slice (serializes as [] not null)")
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("List filters by status", func(t *testing.T) {
		r := newRepo(t)
		pub := catalog.NewShow("p", "p", "", catalog.FormatPodcast, "ar", owner)
		pub.Status = catalog.StatusPublished
		if err := r.Create(ctx, pub); err != nil {
			t.Fatalf("Create pub: %v", err)
		}
		if err := r.Create(ctx, catalog.NewShow("d", "d", "", catalog.FormatPodcast, "ar", owner)); err != nil {
			t.Fatalf("Create draft: %v", err)
		}
		got, err := r.List(ctx, ShowFilter{Status: catalog.StatusPublished})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Slug != "p" {
			t.Errorf("got %d items, want 1 published", len(got))
		}
	})
}

type episodeRepoFactory func(t *testing.T) EpisodeRepository

// ensureShowFunc makes a parent show exist so episodes satisfy the FK.
// Memory passes a no-op; Postgres inserts a shows row.
type ensureShowFunc func(t *testing.T, showID uuid.UUID)

func runEpisodeRepositoryContract(t *testing.T, newRepo episodeRepoFactory, ensureShow ensureShowFunc) {
	t.Helper()
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV7())
	showID := uuid.Must(uuid.NewV7())

	t.Run("Create then GetByID round-trips", func(t *testing.T) {
		r := newRepo(t)
		ensureShow(t, showID)
		ep := catalog.NewEpisode(showID, "حلقة ١", "ep-1", "", 1, catalog.ContentTypeAudio, "ar", 3600, owner)
		if err := r.Create(ctx, ep); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := r.GetByID(ctx, ep.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.EpisodeNumber != 1 || got.ShowID != showID {
			t.Errorf("round-trip mismatch: %+v", got)
		}
	})

	t.Run("duplicate episode_number in same show returns ErrConflict", func(t *testing.T) {
		r := newRepo(t)
		ensureShow(t, showID)
		if err := r.Create(ctx, catalog.NewEpisode(showID, "a", "a", "", 1, catalog.ContentTypeAudio, "ar", 10, owner)); err != nil {
			t.Fatalf("Create a: %v", err)
		}
		if err := r.Create(ctx, catalog.NewEpisode(showID, "b", "b", "", 1, catalog.ContentTypeAudio, "ar", 10, owner)); !errors.Is(err, ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("List on empty returns non-nil slice", func(t *testing.T) {
		r := newRepo(t)
		got, err := r.List(ctx, EpisodeFilter{ShowID: uuid.Must(uuid.NewV7())})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil {
			t.Error("List returned nil, want non-nil empty slice")
		}
	})

	t.Run("List filters by show", func(t *testing.T) {
		r := newRepo(t)
		other := uuid.Must(uuid.NewV7())
		ensureShow(t, showID)
		ensureShow(t, other)
		if err := r.Create(ctx, catalog.NewEpisode(showID, "x", "x", "", 1, catalog.ContentTypeAudio, "ar", 10, owner)); err != nil {
			t.Fatalf("Create x: %v", err)
		}
		if err := r.Create(ctx, catalog.NewEpisode(other, "y", "y", "", 1, catalog.ContentTypeAudio, "ar", 10, owner)); err != nil {
			t.Fatalf("Create y: %v", err)
		}
		got, err := r.List(ctx, EpisodeFilter{ShowID: showID})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d items, want 1 for show", len(got))
		}
	})
}

func runUserRepositoryContract(t *testing.T, newRepo func(t *testing.T) UserRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Create then GetByEmail/GetByID", func(t *testing.T) {
		r := newRepo(t)
		u := auth.NewUser("Admin", "admin@thmanyah.local", "hash", auth.RoleAdmin)
		if err := r.Create(ctx, u); err != nil {
			t.Fatalf("Create: %v", err)
		}
		byEmail, err := r.GetByEmail(ctx, "admin@thmanyah.local")
		if err != nil || byEmail.ID != u.ID {
			t.Fatalf("GetByEmail: err=%v got=%+v", err, byEmail)
		}
		byID, err := r.GetByID(ctx, u.ID)
		if err != nil || byID.Email != u.Email {
			t.Fatalf("GetByID: err=%v got=%+v", err, byID)
		}
	})

	t.Run("duplicate email returns ErrConflict", func(t *testing.T) {
		r := newRepo(t)
		if err := r.Create(ctx, auth.NewUser("A", "dup@x.com", "h", auth.RoleEditor)); err != nil {
			t.Fatalf("Create A: %v", err)
		}
		if err := r.Create(ctx, auth.NewUser("B", "dup@x.com", "h", auth.RoleViewer)); !errors.Is(err, ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("missing email returns ErrNotFound", func(t *testing.T) {
		r := newRepo(t)
		if _, err := r.GetByEmail(ctx, "nobody@x.com"); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
