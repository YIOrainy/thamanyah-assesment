package auth

import (
	"net/http"
	"strings"

	"github.com/yazeedalorainy/thmanyah/internal/server"
)

func (j *JWT) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			server.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		p, err := j.Validate(token)
		if err != nil {
			server.Error(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}

func RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				server.Error(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !p.Can(perm) {
				server.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, prefix) {
		return strings.TrimPrefix(h, prefix)
	}
	return ""
}
