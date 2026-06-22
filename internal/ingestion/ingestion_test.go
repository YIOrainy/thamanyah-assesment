package ingestion

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/store"
)

type fakeImporter struct{ show *ImportedShow }

func (f fakeImporter) Import(context.Context, string) (*ImportedShow, error) { return f.show, nil }

func newTestService(t *testing.T, imps map[string]SourceImporter) *Service {
	shows := store.NewMemoryShowRepository(1)
	eps := store.NewMemoryEpisodeRepository(1)
	refs := store.NewMemoryExternalRefRepository()
	t.Cleanup(shows.Close)
	t.Cleanup(eps.Close)
	t.Cleanup(refs.Close)
	return NewService(imps, shows, eps, refs)
}

func TestServiceImportIsIdempotent(t *testing.T) {
	ctx := context.Background()
	data := &ImportedShow{
		ExternalID: "feed-1", Title: "فنجان", Slug: "finjan", Language: "ar",
		Episodes: []ImportedEpisode{
			{ExternalID: "g1", Title: "حلقة ١", Slug: "g1", EpisodeNumber: 1},
			{ExternalID: "g2", Title: "حلقة ٢", Slug: "g2", EpisodeNumber: 2},
		},
	}
	svc := newTestService(t, map[string]SourceImporter{"fake": fakeImporter{data}})
	actor := uuid.Must(uuid.NewV7())

	r1, err := svc.Run(ctx, "fake", "q", actor)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if r1.ShowsCreated != 1 || r1.EpisodesCreated != 2 {
		t.Fatalf("first run = %+v, want 1 show + 2 episodes created", r1)
	}

	r2, err := svc.Run(ctx, "fake", "q", actor)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if r2.ShowsCreated != 0 || r2.EpisodesCreated != 0 || r2.ShowsUpdated != 1 || r2.EpisodesUpdated != 2 {
		t.Fatalf("second run = %+v, want all updates, no creates (idempotent)", r2)
	}
}

func TestServiceUnknownSource(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Run(context.Background(), "nope", "q", uuid.Must(uuid.NewV7())); !errors.Is(err, ErrUnknownSource) {
		t.Errorf("err = %v, want ErrUnknownSource", err)
	}
}

const sampleRSS = `<?xml version="1.0"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
<channel>
  <title>فنجان</title>
  <description>بودكاست</description>
  <language>ar</language>
  <item>
    <guid>g1</guid>
    <title>قهوة الصباح</title>
    <description>حديث</description>
    <pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate>
    <itunes:duration>3600</itunes:duration>
    <enclosure url="http://example.com/1.mp3"/>
  </item>
</channel>
</rss>`

func TestRSSImporter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	show, err := NewRSSImporter(srv.Client()).Import(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if show.Title != "فنجان" || len(show.Episodes) != 1 {
		t.Fatalf("show = %+v, want فنجان with 1 episode", show)
	}
	ep := show.Episodes[0]
	if ep.ExternalID != "g1" || ep.DurationSeconds != 3600 || ep.PublishedAt.IsZero() {
		t.Errorf("episode = %+v", ep)
	}
}

func TestCSVImporter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finjan.csv")
	content := "external_id,title,description,episode_number,duration_seconds,media_url\n" +
		"g1,قهوة,desc,1,3600,http://x/1.mp3\n" +
		"g2,شاي,desc,2,1800,http://x/2.mp3\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	show, err := NewCSVImporter().Import(context.Background(), path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(show.Episodes) != 2 || show.Episodes[0].ExternalID != "g1" || show.Episodes[1].DurationSeconds != 1800 {
		t.Fatalf("episodes = %+v", show.Episodes)
	}
}
