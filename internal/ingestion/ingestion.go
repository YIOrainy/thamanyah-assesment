package ingestion

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

// ErrUnknownSource is returned when no importer is registered for a source.
var ErrUnknownSource = errors.New("ingestion: unknown source")

// ImportedEpisode is a source-agnostic episode produced by an importer.
type ImportedEpisode struct {
	ExternalID      string
	Title           string
	Description     string
	Slug            string
	EpisodeNumber   int
	DurationSeconds int
	PublishedAt     time.Time
	MediaURL        string
}

// ImportedShow is a source-agnostic show + its episodes.
type ImportedShow struct {
	ExternalID  string
	Title       string
	Slug        string
	Description string
	Language    string
	URL         string
	Episodes    []ImportedEpisode
}

// SourceImporter pulls a show + episodes from one external source. Adding a new
// source = implementing this interface and registering an adapter (open/closed).
type SourceImporter interface {
	Import(ctx context.Context, query string) (*ImportedShow, error)
}

// Result reports what an import run changed.
type Result struct {
	ShowsCreated    int `json:"shows_created"`
	ShowsUpdated    int `json:"shows_updated"`
	EpisodesCreated int `json:"episodes_created"`
	EpisodesUpdated int `json:"episodes_updated"`
}

// Service runs imports through registered SourceImporters and idempotently
// upserts the results into the catalog (deduped via external_refs).
type Service struct {
	importers map[string]SourceImporter
	shows     store.ShowRepository
	episodes  store.EpisodeRepository
	refs      store.ExternalRefRepository
}

func NewService(importers map[string]SourceImporter, shows store.ShowRepository, episodes store.EpisodeRepository, refs store.ExternalRefRepository) *Service {
	return &Service{importers: importers, shows: shows, episodes: episodes, refs: refs}
}

func (s *Service) Run(ctx context.Context, source, query string, actor uuid.UUID) (*Result, error) {
	slog.InfoContext(ctx, "import started", "source", source, "query", query, "actor", actor)
	imp, ok := s.importers[source]
	if !ok {
		slog.WarnContext(ctx, "import rejected", "source", source, "reason", "unknown_source", "actor", actor)
		return nil, ErrUnknownSource
	}
	data, err := imp.Import(ctx, query)
	if err != nil {
		slog.ErrorContext(ctx, "import source fetch failed", "source", source, "query", query, "actor", actor, "error", err)
		return nil, err
	}

	res := &Result{}
	showID, created, err := s.upsertShow(ctx, source, data, actor)
	if err != nil {
		slog.ErrorContext(ctx, "import show upsert failed", "source", source, "external_id", data.ExternalID, "actor", actor, "error", err)
		return nil, err
	}
	if created {
		res.ShowsCreated++
	} else {
		res.ShowsUpdated++
	}

	for _, ep := range data.Episodes {
		created, err := s.upsertEpisode(ctx, source, showID, data.Language, ep, actor)
		if err != nil {
			slog.ErrorContext(ctx, "import episode upsert failed",
				"source", source,
				"show_id", showID,
				"external_id", ep.ExternalID,
				"actor", actor,
				"error", err,
			)
			return nil, err
		}
		if created {
			res.EpisodesCreated++
		} else {
			res.EpisodesUpdated++
		}
	}
	slog.InfoContext(ctx, "import finished",
		"source", source,
		"show_id", showID,
		"actor", actor,
		"shows_created", res.ShowsCreated,
		"shows_updated", res.ShowsUpdated,
		"episodes_created", res.EpisodesCreated,
		"episodes_updated", res.EpisodesUpdated,
	)
	return res, nil
}

func (s *Service) upsertShow(ctx context.Context, source string, d *ImportedShow, actor uuid.UUID) (uuid.UUID, bool, error) {
	if ownerID, found, err := s.refs.FindOwner(ctx, source, store.OwnerShow, d.ExternalID); err != nil {
		return uuid.Nil, false, err
	} else if found {
		show, err := s.shows.GetByID(ctx, ownerID)
		if err != nil {
			return uuid.Nil, false, err
		}
		show.Title = d.Title
		show.Description = d.Description
		show.UpdatedBy = actor
		show.UpdatedAt = time.Now().UTC()
		if err := s.shows.Update(ctx, show); err != nil {
			return uuid.Nil, false, err
		}
		return ownerID, false, nil
	}

	lang := d.Language
	if lang == "" {
		lang = "ar"
	}
	show := catalog.NewShow(d.Title, d.Slug, d.Description, catalog.FormatPodcast, lang, actor)
	show.Status = catalog.StatusPublished // imported content is already live on the source
	if err := s.shows.Create(ctx, show); err != nil {
		return uuid.Nil, false, err
	}
	ref := &store.ExternalRef{
		ID: uuid.Must(uuid.NewV7()), Source: source, OwnerType: store.OwnerShow,
		OwnerID: show.ID, ExternalID: d.ExternalID, ExternalURL: d.URL, ImportedAt: time.Now().UTC(),
	}
	if err := s.refs.Create(ctx, ref); err != nil {
		return uuid.Nil, false, err
	}
	return show.ID, true, nil
}

func (s *Service) upsertEpisode(ctx context.Context, source string, showID uuid.UUID, lang string, d ImportedEpisode, actor uuid.UUID) (bool, error) {
	if ownerID, found, err := s.refs.FindOwner(ctx, source, store.OwnerEpisode, d.ExternalID); err != nil {
		return false, err
	} else if found {
		ep, err := s.episodes.GetByID(ctx, ownerID)
		if err != nil {
			return false, err
		}
		ep.Title = d.Title
		ep.Description = d.Description
		ep.DurationSeconds = d.DurationSeconds
		ep.UpdatedBy = actor
		ep.UpdatedAt = time.Now().UTC()
		if err := s.episodes.Update(ctx, ep); err != nil {
			return false, err
		}
		return false, nil
	}

	if lang == "" {
		lang = "ar"
	}
	ep := catalog.NewEpisode(showID, d.Title, d.Slug, d.Description, d.EpisodeNumber, catalog.ContentTypeAudio, lang, d.DurationSeconds, actor)
	ep.Status = catalog.StatusPublished // imported content is already live on the source
	if !d.PublishedAt.IsZero() {
		pt := d.PublishedAt
		ep.PublishedAt = &pt
	}
	if err := s.episodes.Create(ctx, ep); err != nil {
		return false, err
	}
	ref := &store.ExternalRef{
		ID: uuid.Must(uuid.NewV7()), Source: source, OwnerType: store.OwnerEpisode,
		OwnerID: ep.ID, ExternalID: d.ExternalID, ExternalURL: d.MediaURL, ImportedAt: time.Now().UTC(),
	}
	if err := s.refs.Create(ctx, ref); err != nil {
		return false, err
	}
	return true, nil
}
