package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

func TestShowOffsetPagination(t *testing.T) {
	ctx := context.Background()
	r := NewMemoryShowRepository(2)
	t.Cleanup(r.Close)
	owner := uuid.Must(uuid.NewV7())

	base := time.Now().UTC()
	for i := 0; i < 5; i++ {
		s := catalog.NewShow(fmt.Sprintf("show %d", i), fmt.Sprintf("slug-%d", i), "", catalog.FormatPodcast, "ar", owner)
		s.CreatedAt = base.Add(time.Duration(i) * time.Second)
		if err := r.Create(ctx, s); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	total, err := r.Count(ctx, ShowFilter{})
	if err != nil || total != 5 {
		t.Fatalf("Count = %d (err %v), want 5", total, err)
	}

	seen := map[string]bool{}
	for _, off := range []int{0, 2, 4} {
		page, err := r.List(ctx, ShowFilter{Limit: 2, Offset: off})
		if err != nil {
			t.Fatalf("List offset %d: %v", off, err)
		}
		for _, s := range page {
			if seen[s.Slug] {
				t.Errorf("offset paging returned duplicate %s", s.Slug)
			}
			seen[s.Slug] = true
		}
	}
	if len(seen) != 5 {
		t.Errorf("offset paging covered %d shows, want 5", len(seen))
	}
}

func TestSearchKeysetPagination(t *testing.T) {
	ctx := context.Background()
	r := NewMemoryEpisodeRepository(2)
	t.Cleanup(r.Close)
	owner := uuid.Must(uuid.NewV7())
	showID := uuid.Must(uuid.NewV7())

	base := time.Now().UTC()
	for i := 0; i < 5; i++ {
		e := catalog.NewEpisode(showID, fmt.Sprintf("ep %d", i), fmt.Sprintf("ep-%d", i), "قهوة", i+1, catalog.ContentTypeAudio, "ar", 10, owner)
		e.Status = catalog.StatusPublished
		pt := base.Add(time.Duration(i) * time.Second)
		e.PublishedAt = &pt
		if err := r.Create(ctx, e); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	seen := map[string]bool{}
	cursor := ""
	for pages := 0; ; pages++ {
		got, err := r.SearchEpisodes(ctx, "قهوة", SearchFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		for _, e := range got {
			if seen[e.Slug] {
				t.Errorf("keyset paging returned duplicate %s", e.Slug)
			}
			seen[e.Slug] = true
		}
		if len(got) < 2 {
			break
		}
		last := got[len(got)-1]
		cursor = EncodeCursor(*last.PublishedAt, last.ID)
		if pages > 10 {
			t.Fatal("keyset paging did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Errorf("keyset paging covered %d episodes, want 5", len(seen))
	}
}
