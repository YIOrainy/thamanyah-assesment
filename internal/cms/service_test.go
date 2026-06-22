package cms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

func newTestService(t *testing.T) *service {
	shows := store.NewMemoryShowRepository(1)
	eps := store.NewMemoryEpisodeRepository(1)
	users := store.NewMemoryUserRepository()
	refs := store.NewMemoryExternalRefRepository()
	t.Cleanup(shows.Close)
	t.Cleanup(eps.Close)
	t.Cleanup(users.Close)
	t.Cleanup(refs.Close)
	imp := ingestion.NewService(nil, shows, eps, refs)
	return newService(shows, eps, users, auth.NewJWT("test-secret", time.Hour), imp)
}

func TestService_CreatePublishShow(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	actor := uuid.Must(uuid.NewV7())

	show, err := svc.createShow(ctx, createShowInput{
		Title: "فنجان", Slug: "finjan", Format: catalog.FormatPodcast, Language: "ar",
	}, actor)
	if err != nil {
		t.Fatalf("createShow: %v", err)
	}
	if show.Status != catalog.StatusDraft {
		t.Errorf("new show status = %q, want draft", show.Status)
	}

	published, err := svc.publishShow(ctx, show.ID, actor)
	if err != nil {
		t.Fatalf("publishShow: %v", err)
	}
	if published.Status != catalog.StatusPublished {
		t.Errorf("status = %q, want published", published.Status)
	}
}

func TestService_CreateShowValidation(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.createShow(context.Background(), createShowInput{
		Title: "x", Slug: "x", Format: "bogus", Language: "ar",
	}, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, errValidation) {
		t.Errorf("err = %v, want errValidation", err)
	}
}

func TestService_ListEpisodesMissingShow(t *testing.T) {
	svc := newTestService(t)
	_, _, err := svc.listEpisodes(context.Background(), uuid.Must(uuid.NewV7()), 1, 20)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound (missing show → 404)", err)
	}
}

func TestService_CreateEpisodeRequiresShow(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.createEpisode(context.Background(), uuid.Must(uuid.NewV7()), createEpisodeInput{
		Title: "ep", Slug: "ep-1", ContentType: catalog.ContentTypeAudio, Language: "ar", EpisodeNumber: 1,
	}, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound (missing parent show)", err)
	}
}
