package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is a CMS user (identity). PasswordHash is a bcrypt hash; never serialized.
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewUser creates a user with a fresh time-ordered ID. password must already be hashed.
func NewUser(name, email, passwordHash string, role Role) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.Must(uuid.NewV7()),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
