package discovery

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yazeedalorainy/thmanyah/internal/cache"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

const (
	APIVersion = "v1"
	APIPrefix  = "/api/" + APIVersion
)

func NewRouter(shows store.ShowRepository, episodes store.EpisodeRepository, search store.Searcher, c cache.Cache, ttl time.Duration) http.Handler {
	r := chi.NewRouter()
	r.Mount(APIPrefix, newHandlers(newService(shows, episodes, search, c, ttl)).routes())
	return r
}

func (h *handlers) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/search", h.handleSearch)
	r.Route("/shows", func(r chi.Router) {
		r.Get("/", h.handleBrowseShows)
		r.Get("/{slug}", h.handleGetShow)
		r.Get("/{slug}/episodes", h.handleShowEpisodes)
	})
	r.Get("/episodes/{slug}", h.handleGetEpisode)
	return r
}
