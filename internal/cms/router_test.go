package cms

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/ingestion"
	"github.com/yazeedalorainy/thmanyah/internal/store"
)

func newTestRouter(t *testing.T) (http.Handler, *auth.JWT) {
	shows := store.NewMemoryShowRepository(1)
	episodes := store.NewMemoryEpisodeRepository(1)
	users := store.NewMemoryUserRepository()
	refs := store.NewMemoryExternalRefRepository()
	t.Cleanup(shows.Close)
	t.Cleanup(episodes.Close)
	t.Cleanup(users.Close)
	t.Cleanup(refs.Close)
	jwt := auth.NewJWT("test-secret", time.Hour)
	imp := ingestion.NewService(nil, shows, episodes, refs)
	return NewRouter(shows, episodes, users, jwt, imp), jwt
}

func TestRouterMountsCurrentAPIVersion(t *testing.T) {
	router, jwt := newTestRouter(t)
	token, _ := jwt.Issue(uuid.Must(uuid.NewV7()), auth.RoleAdmin, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, APIPrefix+"/shows", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s (admin) status = %d, want 200", req.URL.Path, rec.Code)
	}
}

func TestRouterRequiresAuth(t *testing.T) {
	router, _ := newTestRouter(t)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/shows", nil)) // no token
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /shows without token status = %d, want 401", rec.Code)
	}
}

func TestRouterRejectsUnsupportedAPIVersions(t *testing.T) {
	router, _ := newTestRouter(t)
	for _, path := range []string{"/shows", "/api/v2/shows"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("GET %s status = %d, want 404", path, rec.Code)
			}
		})
	}
}

func TestRouterValidatesBody(t *testing.T) {
	router, jwt := newTestRouter(t)
	token, _ := jwt.Issue(uuid.Must(uuid.NewV7()), auth.RoleAdmin, nil)

	// missing title (required) + invalid format (oneof) → 400 at the DTO boundary
	body := strings.NewReader(`{"slug":"x","format":"bogus","language":"ar"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/shows", body)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /shows invalid body status = %d, want 400", rec.Code)
	}
}

func TestRouterForbidsViewerWrite(t *testing.T) {
	router, jwt := newTestRouter(t)
	token, _ := jwt.Issue(uuid.Must(uuid.NewV7()), auth.RoleViewer, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, APIPrefix+"/shows", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer POST /shows status = %d, want 403", rec.Code)
	}
}
