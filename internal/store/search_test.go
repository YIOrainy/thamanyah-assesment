package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

type searchSetup struct {
	repo       EpisodeRepository
	search     Searcher
	ensureShow ensureShowFunc
}

// runSearcherContract seeds one published + one draft episode, then verifies
// search behaviour. Setup runs once; subtests are read-only queries.
func runSearcherContract(t *testing.T, mk func(t *testing.T) searchSetup) {
	t.Helper()
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV7())

	s := mk(t)
	showID := uuid.Must(uuid.NewV7())
	s.ensureShow(t, showID)

	now := time.Now().UTC()
	pub := catalog.NewEpisode(showID, "قهوة الصباح", "ep-1", "حديث عن القهوة", 1, catalog.ContentTypeAudio, "ar", 10, owner)
	pub.Status = catalog.StatusPublished
	pub.PublishedAt = &now
	if err := s.repo.Create(ctx, pub); err != nil {
		t.Fatalf("create published: %v", err)
	}
	draft := catalog.NewEpisode(showID, "شاي بالنعناع", "ep-2", "", 2, catalog.ContentTypeAudio, "ar", 10, owner)
	if err := s.repo.Create(ctx, draft); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	t.Run("matches a published episode by term", func(t *testing.T) {
		got, err := s.search.SearchEpisodes(ctx, "قهوة", SearchFilter{})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(got) != 1 || got[0].Slug != "ep-1" {
			t.Errorf("got %d results, want 1 (ep-1)", len(got))
		}
	})

	t.Run("excludes drafts", func(t *testing.T) {
		got, err := s.search.SearchEpisodes(ctx, "شاي", SearchFilter{})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d results, want 0 (draft excluded)", len(got))
		}
	})

	t.Run("empty query returns published only", func(t *testing.T) {
		got, err := s.search.SearchEpisodes(ctx, "", SearchFilter{})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d results, want 1 published", len(got))
		}
	})
}

func TestMemorySearcher(t *testing.T) {
	runSearcherContract(t, func(t *testing.T) searchSetup {
		r := NewMemoryEpisodeRepository(1)
		t.Cleanup(r.Close)
		return searchSetup{repo: r, search: r, ensureShow: func(t *testing.T, id uuid.UUID) {}}
	})
}
