package discovery

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/server"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

const maxLimit = 100

type handlers struct {
	svc *service
}

func newHandlers(svc *service) *handlers { return &handlers{svc: svc} }

// cursorResponse is the Discovery keyset-pagination envelope.
type cursorResponse struct {
	Items      any    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func discoveryError(err error) (int, string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, store.ErrInvalidCursor):
		return http.StatusBadRequest, "invalid cursor"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	status, detail := discoveryError(err)
	attrs := []any{
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	}
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "discovery request failed", attrs...)
	} else {
		slog.WarnContext(r.Context(), "discovery request failed", attrs...)
	}
	server.Error(w, status, detail)
}

// resolveLimit reads ?limit= with a default and a hard cap.
func resolveLimit(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func (h *handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := resolveLimit(r)
	eps, err := h.svc.searchEpisodes(r.Context(), q.Get("q"), store.SearchFilter{
		Language: q.Get("language"),
		Limit:    limit,
		Cursor:   q.Get("cursor"),
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, cursorResponse{Items: eps, NextCursor: episodesCursor(eps, limit)})
}

func (h *handlers) handleBrowseShows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := resolveLimit(r)
	shows, err := h.svc.browseShows(r.Context(), store.ShowFilter{
		Format:   catalog.Format(q.Get("format")),
		Language: q.Get("language"),
		Limit:    limit,
		Cursor:   q.Get("cursor"),
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, cursorResponse{Items: shows, NextCursor: showsCursor(shows, limit)})
}

func (h *handlers) handleGetShow(w http.ResponseWriter, r *http.Request) {
	show, err := h.svc.getShow(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, show)
}

func (h *handlers) handleShowEpisodes(w http.ResponseWriter, r *http.Request) {
	eps, err := h.svc.listShowEpisodes(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, cursorResponse{Items: eps})
}

func (h *handlers) handleGetEpisode(w http.ResponseWriter, r *http.Request) {
	ep, err := h.svc.getEpisode(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, ep)
}

// episodesCursor returns the next-page cursor, or "" when there's likely no more.
func episodesCursor(eps []*catalog.Episode, limit int) string {
	if len(eps) == 0 || len(eps) < limit {
		return ""
	}
	last := eps[len(eps)-1]
	if last.PublishedAt == nil {
		return ""
	}
	return store.EncodeCursor(*last.PublishedAt, last.ID)
}

func showsCursor(shows []*catalog.Show, limit int) string {
	if len(shows) == 0 || len(shows) < limit {
		return ""
	}
	last := shows[len(shows)-1]
	return store.EncodeCursor(last.CreatedAt, last.ID)
}
