package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "s3cret") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	uid := uuid.Must(uuid.NewV7())

	token, err := j.Issue(uid, RoleEditor, nil) // nil scope = full role permissions
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	p, err := j.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if p.UserID != uid || p.Role != RoleEditor {
		t.Errorf("principal mismatch: %+v", p)
	}
	if !p.Can(PermShowsPublish) {
		t.Error("editor should have shows:publish")
	}
	if p.Can(PermUsersManage) {
		t.Error("editor should NOT have users:manage")
	}
}

func TestJWTScopeNarrows(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	// admin role, but token scoped to only imports:run
	token, _ := j.Issue(uuid.Must(uuid.NewV7()), RoleAdmin, []Permission{PermImportsRun})
	p, err := j.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !p.Can(PermImportsRun) {
		t.Error("scoped permission should be allowed")
	}
	if p.Can(PermUsersManage) {
		t.Error("scope must narrow: users:manage not in scope")
	}
}

func TestValidateRejectsTampered(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	if _, err := j.Validate("not.a.token"); err == nil {
		t.Error("expected error for garbage token")
	}
}

func TestMiddleware(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	// chain: Authenticate -> RequirePermission(shows:write) -> ok
	handler := j.Authenticate(RequirePermission(PermShowsWrite)(http.HandlerFunc(ok)))

	t.Run("no token -> 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("code = %d, want 401", rec.Code)
		}
	})

	t.Run("viewer -> 403", func(t *testing.T) {
		token, _ := j.Issue(uuid.Must(uuid.NewV7()), RoleViewer, nil)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("code = %d, want 403", rec.Code)
		}
	})

	t.Run("editor -> 200", func(t *testing.T) {
		token, _ := j.Issue(uuid.Must(uuid.NewV7()), RoleEditor, nil)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("code = %d, want 200", rec.Code)
		}
	})
}
