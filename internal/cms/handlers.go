package cms

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
	"github.com/yazeedalorainy/thmanyah/internal/server"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

type handlers struct {
	svc *service
}

func newHandlers(svc *service) *handlers { return &handlers{svc: svc} }

// actor returns the authenticated user id from the request context (set by the
// Authenticate middleware).
func actor(r *http.Request) uuid.UUID {
	if p, ok := auth.PrincipalFrom(r.Context()); ok {
		return p.UserID
	}
	return uuid.Nil
}

func cmsError(err error) (int, string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, store.ErrConflict):
		return http.StatusConflict, "already exists"
	case errors.Is(err, errValidation):
		return http.StatusBadRequest, "validation failed"
	case errors.Is(err, errInvalidCredentials):
		return http.StatusUnauthorized, "invalid credentials"
	case errors.Is(err, ingestion.ErrUnknownSource):
		return http.StatusBadRequest, "unknown import source"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	status, detail := cmsError(err)
	attrs := []any{
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
		"actor", actor(r),
		"error", err,
	}
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "cms request failed", attrs...)
	} else {
		slog.WarnContext(r.Context(), "cms request failed", attrs...)
	}
	server.Error(w, status, detail)
}

func (h *handlers) handleRunImport(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if !server.DecodeJSON(w, r, &req) {
		return
	}
	result, err := h.svc.runImport(r.Context(), req.Source, req.Query, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, result)
}

func (h *handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !server.DecodeJSON(w, r, &req) {
		return
	}
	token, err := h.svc.login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, loginResponse{AccessToken: token})
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// pageParams reads ?page= and ?page_size= with sane defaults and a cap.
func pageParams(r *http.Request) (page, pageSize int) {
	page, pageSize = 1, defaultPageSize
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = n
	}
	if n, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && n > 0 {
		pageSize = n
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func parseID(w http.ResponseWriter, r *http.Request, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		slog.WarnContext(r.Context(), "invalid path id",
			"method", r.Method,
			"path", r.URL.Path,
			"value", raw,
			"error", err,
		)
		server.Error(w, http.StatusBadRequest, "invalid id")
		return uuid.Nil, false
	}
	return id, true
}

// --- shows ---

func (h *handlers) handleCreateShow(w http.ResponseWriter, r *http.Request) {
	var req createShowRequest
	if !server.DecodeJSON(w, r, &req) {
		return
	}
	show, err := h.svc.createShow(r.Context(), createShowInput{
		Title: req.Title, Slug: req.Slug, Description: req.Description,
		Format: catalog.Format(req.Format), Language: req.Language,
	}, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusCreated, show)
}

func (h *handlers) handleListShows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.ShowFilter{
		Format:   catalog.Format(q.Get("format")),
		Status:   catalog.Status(q.Get("status")),
		Language: q.Get("language"),
	}
	page, pageSize := pageParams(r)
	shows, total, err := h.svc.listShows(r.Context(), f, page, pageSize)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, pagedResponse{Items: shows, Page: page, PageSize: pageSize, Total: total})
}

func (h *handlers) handleGetShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, chi.URLParam(r, "showID"))
	if !ok {
		return
	}
	show, err := h.svc.getShow(r.Context(), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, show)
}

func (h *handlers) handleUpdateShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, chi.URLParam(r, "showID"))
	if !ok {
		return
	}
	var req updateShowRequest
	if !server.DecodeJSON(w, r, &req) {
		return
	}
	in := updateShowInput{Title: req.Title, Description: req.Description, Language: req.Language}
	if req.Format != nil {
		f := catalog.Format(*req.Format)
		in.Format = &f
	}
	show, err := h.svc.updateShow(r.Context(), id, in, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, show)
}

func (h *handlers) handlePublishShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, chi.URLParam(r, "showID"))
	if !ok {
		return
	}
	show, err := h.svc.publishShow(r.Context(), id, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, show)
}

// --- episodes ---

func (h *handlers) handleCreateEpisode(w http.ResponseWriter, r *http.Request) {
	showID, ok := parseID(w, r, chi.URLParam(r, "showID"))
	if !ok {
		return
	}
	var req createEpisodeRequest
	if !server.DecodeJSON(w, r, &req) {
		return
	}
	ep, err := h.svc.createEpisode(r.Context(), showID, createEpisodeInput{
		Title: req.Title, Slug: req.Slug, Description: req.Description,
		EpisodeNumber: req.EpisodeNumber, ContentType: catalog.ContentType(req.ContentType),
		Language: req.Language, DurationSeconds: req.DurationSeconds,
	}, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusCreated, ep)
}

func (h *handlers) handleListEpisodes(w http.ResponseWriter, r *http.Request) {
	showID, ok := parseID(w, r, chi.URLParam(r, "showID"))
	if !ok {
		return
	}
	page, pageSize := pageParams(r)
	eps, total, err := h.svc.listEpisodes(r.Context(), showID, page, pageSize)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, pagedResponse{Items: eps, Page: page, PageSize: pageSize, Total: total})
}

func (h *handlers) handleGetEpisode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, chi.URLParam(r, "episodeID"))
	if !ok {
		return
	}
	ep, err := h.svc.getEpisode(r.Context(), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, ep)
}

func (h *handlers) handlePublishEpisode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, chi.URLParam(r, "episodeID"))
	if !ok {
		return
	}
	ep, err := h.svc.publishEpisode(r.Context(), id, actor(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, ep)
}
