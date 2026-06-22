package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/cache"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

func newSeededRouter(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV7())

	shows := store.NewMemoryShowRepository(1)
	episodes := store.NewMemoryEpisodeRepository(1)
	t.Cleanup(shows.Close)
	t.Cleanup(episodes.Close)

	show := catalog.NewShow("فنجان", "finjan", "بودكاست", catalog.FormatPodcast, "ar", owner)
	show.Status = catalog.StatusPublished
	if err := shows.Create(ctx, show); err != nil {
		t.Fatalf("seed show: %v", err)
	}
	now := time.Now().UTC()
	ep := catalog.NewEpisode(show.ID, "قهوة الصباح", "ep-1", "حديث عن القهوة", 1, catalog.ContentTypeAudio, "ar", 10, owner)
	ep.Status = catalog.StatusPublished
	ep.PublishedAt = &now
	if err := episodes.Create(ctx, ep); err != nil {
		t.Fatalf("seed episode: %v", err)
	}

	return NewRouter(shows, episodes, episodes, cache.NewMemory(), time.Minute)
}

func get(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestDiscoverySearchUnderAPIPrefix(t *testing.T) {
	router := newSeededRouter(t)

	rec := get(t, router, APIPrefix+"/search?q=قهوة")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Items []catalog.Episode `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Slug != "ep-1" {
		t.Errorf("got %d results, want 1 (ep-1)", len(body.Items))
	}
}

func TestDiscoveryRejectsUnsupportedVersion(t *testing.T) {
	router := newSeededRouter(t)
	if rec := get(t, router, "/api/v2/search"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestDiscoveryUnknownShowIs404(t *testing.T) {
	router := newSeededRouter(t)
	if rec := get(t, router, APIPrefix+"/shows/does-not-exist"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
