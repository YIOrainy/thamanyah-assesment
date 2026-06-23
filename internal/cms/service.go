package cms

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

var errInvalidCredentials = errors.New("cms: invalid credentials")

// service holds the CMS write-side use cases over the catalog repositories.
type service struct {
	shows    store.ShowRepository
	episodes store.EpisodeRepository
	users    store.UserRepository
	jwt      *auth.JWT
	imports  *ingestion.Service
}

func newService(shows store.ShowRepository, episodes store.EpisodeRepository, users store.UserRepository, jwt *auth.JWT, imports *ingestion.Service) *service {
	return &service{shows: shows, episodes: episodes, users: users, jwt: jwt, imports: imports}
}

func (s *service) runImport(ctx context.Context, source, query string, actor uuid.UUID) (*ingestion.Result, error) {
	result, err := s.imports.Run(ctx, source, query, actor)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "cms import completed",
		"source", source,
		"actor", actor,
		"shows_created", result.ShowsCreated,
		"shows_updated", result.ShowsUpdated,
		"episodes_created", result.EpisodesCreated,
		"episodes_updated", result.EpisodesUpdated,
	)
	return result, nil
}

// login verifies credentials and issues a JWT. It returns errInvalidCredentials
// for both unknown email and wrong password (no user enumeration).
func (s *service) login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		slog.WarnContext(ctx, "cms login failed", "email", email, "reason", "user_not_found")
		return "", errInvalidCredentials
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		slog.WarnContext(ctx, "cms login failed", "email", email, "user_id", u.ID, "reason", "bad_password")
		return "", errInvalidCredentials
	}
	token, err := s.jwt.Issue(u.ID, u.Role, nil) // nil scope = role's full permission set
	if err != nil {
		slog.ErrorContext(ctx, "cms token issue failed", "email", email, "user_id", u.ID, "role", u.Role, "error", err)
		return "", err
	}
	slog.InfoContext(ctx, "cms login succeeded", "email", email, "user_id", u.ID, "role", u.Role)
	return token, nil
}

type createShowInput struct {
	Title       string
	Slug        string
	Description string
	Format      catalog.Format
	Language    string
}

func (s *service) createShow(ctx context.Context, in createShowInput, actor uuid.UUID) (*catalog.Show, error) {
	show := catalog.NewShow(in.Title, in.Slug, in.Description, in.Format, in.Language, actor)
	if err := s.shows.Create(ctx, show); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "show created",
		"show_id", show.ID,
		"slug", show.Slug,
		"format", show.Format,
		"language", show.Language,
		"actor", actor,
	)
	return show, nil
}

type updateShowInput struct {
	Title       *string
	Description *string
	Format      *catalog.Format
	Language    *string
}

func (s *service) updateShow(ctx context.Context, id uuid.UUID, in updateShowInput, actor uuid.UUID) (*catalog.Show, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		show.Title = *in.Title
	}
	if in.Description != nil {
		show.Description = *in.Description
	}
	if in.Language != nil {
		show.Language = *in.Language
	}
	if in.Format != nil {
		show.Format = *in.Format
	}
	show.UpdatedBy = actor
	show.UpdatedAt = time.Now().UTC()
	if err := s.shows.Update(ctx, show); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "show updated",
		"show_id", show.ID,
		"slug", show.Slug,
		"status", show.Status,
		"actor", actor,
	)
	return show, nil
}

func (s *service) publishShow(ctx context.Context, id, actor uuid.UUID) (*catalog.Show, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	show.Status = catalog.StatusPublished
	show.UpdatedBy = actor
	show.UpdatedAt = time.Now().UTC()
	if err := s.shows.Update(ctx, show); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "show published", "show_id", show.ID, "slug", show.Slug, "actor", actor)
	return show, nil
}

func (s *service) getShow(ctx context.Context, id uuid.UUID) (*catalog.Show, error) {
	return s.shows.GetByID(ctx, id)
}

func (s *service) listShows(ctx context.Context, f store.ShowFilter, page, pageSize int) ([]*catalog.Show, int, error) {
	f.Limit = pageSize
	f.Offset = (page - 1) * pageSize
	items, err := s.shows.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.shows.Count(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type createEpisodeInput struct {
	Title           string
	Slug            string
	Description     string
	EpisodeNumber   int
	ContentType     catalog.ContentType
	Language        string
	DurationSeconds int
}

func (s *service) createEpisode(ctx context.Context, showID uuid.UUID, in createEpisodeInput, actor uuid.UUID) (*catalog.Episode, error) {
	if _, err := s.shows.GetByID(ctx, showID); err != nil {
		return nil, err // ErrNotFound if the parent show is missing
	}
	ep := catalog.NewEpisode(showID, in.Title, in.Slug, in.Description, in.EpisodeNumber, in.ContentType, in.Language, in.DurationSeconds, actor)
	if err := s.episodes.Create(ctx, ep); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "episode created",
		"episode_id", ep.ID,
		"show_id", showID,
		"slug", ep.Slug,
		"episode_number", ep.EpisodeNumber,
		"content_type", ep.ContentType,
		"actor", actor,
	)
	return ep, nil
}

func (s *service) listEpisodes(ctx context.Context, showID uuid.UUID, page, pageSize int) ([]*catalog.Episode, int, error) {
	if _, err := s.shows.GetByID(ctx, showID); err != nil {
		return nil, 0, err // ErrNotFound → 404; distinguishes "missing show" from "no episodes"
	}
	f := store.EpisodeFilter{ShowID: showID, Limit: pageSize, Offset: (page - 1) * pageSize}
	items, err := s.episodes.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.episodes.Count(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *service) getEpisode(ctx context.Context, id uuid.UUID) (*catalog.Episode, error) {
	return s.episodes.GetByID(ctx, id)
}

func (s *service) publishEpisode(ctx context.Context, id, actor uuid.UUID) (*catalog.Episode, error) {
	ep, err := s.episodes.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ep.Status = catalog.StatusPublished
	ep.PublishedAt = &now
	ep.UpdatedBy = actor
	ep.UpdatedAt = now
	if err := s.episodes.Update(ctx, ep); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "episode published",
		"episode_id", ep.ID,
		"show_id", ep.ShowID,
		"slug", ep.Slug,
		"actor", actor,
	)
	return ep, nil
}
