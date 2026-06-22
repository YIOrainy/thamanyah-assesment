package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/yazeedalorainy/thmanyah/internal/cache"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

const defaultLimit = 20

// service holds the public read-side use cases. Reads are cache-aside (Redis in
// prod, noop when disabled) and restricted to published content.
type service struct {
	shows    store.ShowRepository
	episodes store.EpisodeRepository
	search   store.Searcher
	cache    cache.Cache
	ttl      time.Duration
}

func newService(shows store.ShowRepository, episodes store.EpisodeRepository, search store.Searcher, c cache.Cache, ttl time.Duration) *service {
	return &service{shows: shows, episodes: episodes, search: search, cache: c, ttl: ttl}
}

func (s *service) searchEpisodes(ctx context.Context, query string, f store.SearchFilter) ([]*catalog.Episode, error) {
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	key := fmt.Sprintf("disc:search:q=%s:lang=%s:limit=%d:cursor=%s", query, f.Language, f.Limit, f.Cursor)
	return cache.Remember(ctx, s.cache, key, s.ttl, func(ctx context.Context) ([]*catalog.Episode, error) {
		return s.search.SearchEpisodes(ctx, query, f)
	})
}

func (s *service) browseShows(ctx context.Context, f store.ShowFilter) ([]*catalog.Show, error) {
	f.Status = catalog.StatusPublished // public sees published only
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	key := fmt.Sprintf("disc:shows:format=%s:lang=%s:limit=%d:cursor=%s", f.Format, f.Language, f.Limit, f.Cursor)
	return cache.Remember(ctx, s.cache, key, s.ttl, func(ctx context.Context) ([]*catalog.Show, error) {
		return s.shows.List(ctx, f)
	})
}

func (s *service) getShow(ctx context.Context, slug string) (*catalog.Show, error) {
	return cache.Remember(ctx, s.cache, "disc:show:"+slug, s.ttl, func(ctx context.Context) (*catalog.Show, error) {
		show, err := s.shows.GetBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if show.Status != catalog.StatusPublished {
			return nil, store.ErrNotFound // don't expose drafts to the public
		}
		return show, nil
	})
}

func (s *service) listShowEpisodes(ctx context.Context, slug string) ([]*catalog.Episode, error) {
	show, err := s.getShow(ctx, slug)
	if err != nil {
		return nil, err
	}
	return cache.Remember(ctx, s.cache, "disc:show-episodes:"+slug, s.ttl, func(ctx context.Context) ([]*catalog.Episode, error) {
		return s.episodes.List(ctx, store.EpisodeFilter{ShowID: show.ID, Status: catalog.StatusPublished})
	})
}

func (s *service) getEpisode(ctx context.Context, slug string) (*catalog.Episode, error) {
	return cache.Remember(ctx, s.cache, "disc:episode:"+slug, s.ttl, func(ctx context.Context) (*catalog.Episode, error) {
		ep, err := s.episodes.GetBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if ep.Status != catalog.StatusPublished {
			return nil, store.ErrNotFound
		}
		return ep, nil
	})
}
