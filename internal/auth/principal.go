package auth

import (
	"context"

	"github.com/google/uuid"
)

// Principal is the authenticated caller and its effective permissions.
type Principal struct {
	UserID      uuid.UUID
	Role        Role
	Permissions map[Permission]bool // effective = role permissions ∩ token scopes
}

func (p *Principal) Can(perm Permission) bool {
	return p != nil && p.Permissions[perm]
}

// newPrincipal computes effective permissions. An empty scope means the token
// carries the role's full permission set; a non-empty scope narrows it.
func newPrincipal(userID uuid.UUID, role Role, scopes []Permission) *Principal {
	allowed := make(map[Permission]bool)
	for _, p := range PermissionsForRole(role) {
		allowed[p] = true
	}
	effective := make(map[Permission]bool)
	if len(scopes) == 0 {
		effective = allowed
	} else {
		for _, p := range scopes {
			if allowed[p] {
				effective[p] = true // intersection: a token can only narrow, never widen
			}
		}
	}
	return &Principal{UserID: userID, Role: role, Permissions: effective}
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok
}
