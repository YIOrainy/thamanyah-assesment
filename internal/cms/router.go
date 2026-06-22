package cms

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

const (
	APIVersion = "v1"
	APIPrefix  = "/api/" + APIVersion
)

// NewRouter builds the CMS write-side routes (under APIPrefix). Login is public;
// every other route requires authentication plus a specific permission.
func NewRouter(shows store.ShowRepository, episodes store.EpisodeRepository, users store.UserRepository, jwt *auth.JWT, imports *ingestion.Service) http.Handler {
	h := newHandlers(newService(shows, episodes, users, jwt, imports))
	r := chi.NewRouter()
	r.Mount(APIPrefix, h.routes(jwt))
	return r
}

func (h *handlers) routes(jwt *auth.JWT) chi.Router {
	r := chi.NewRouter()

	r.Post("/auth/login", h.handleLogin) // public

	r.Group(func(r chi.Router) {
		r.Use(jwt.Authenticate)

		r.Route("/shows", func(r chi.Router) {
			r.With(auth.RequirePermission(auth.PermShowsWrite)).Post("/", h.handleCreateShow)
			r.With(auth.RequirePermission(auth.PermShowsRead)).Get("/", h.handleListShows)
			r.Route("/{showID}", func(r chi.Router) {
				r.With(auth.RequirePermission(auth.PermShowsRead)).Get("/", h.handleGetShow)
				r.With(auth.RequirePermission(auth.PermShowsWrite)).Patch("/", h.handleUpdateShow)
				r.With(auth.RequirePermission(auth.PermShowsPublish)).Post("/publish", h.handlePublishShow)
				r.With(auth.RequirePermission(auth.PermEpisodesWrite)).Post("/episodes", h.handleCreateEpisode)
				r.With(auth.RequirePermission(auth.PermEpisodesRead)).Get("/episodes", h.handleListEpisodes)
			})
		})

		r.Route("/episodes", func(r chi.Router) {
			r.With(auth.RequirePermission(auth.PermEpisodesRead)).Get("/{episodeID}", h.handleGetEpisode)
			r.With(auth.RequirePermission(auth.PermEpisodesPublish)).Post("/{episodeID}/publish", h.handlePublishEpisode)
		})

		r.With(auth.RequirePermission(auth.PermImportsRun)).Post("/imports", h.handleRunImport)
	})

	return r
}
