package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Role  string   `json:"role"`
	Scope []string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

func (j *JWT) Issue(userID uuid.UUID, role Role, scopes []Permission) (string, error) {
	now := time.Now()
	claims := Claims{
		Role:  string(role),
		Scope: permsToStrings(scopes),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

func (j *JWT) Validate(tokenStr string) (*Principal, error) {
	var claims Claims
	tok, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid subject: %w", err)
	}
	return newPrincipal(userID, Role(claims.Role), stringsToPerms(claims.Scope)), nil
}

func permsToStrings(perms []Permission) []string {
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}

func stringsToPerms(ss []string) []Permission {
	out := make([]Permission, len(ss))
	for i, s := range ss {
		out[i] = Permission(s)
	}
	return out
}
