package catalog

import (
	"time"

	"github.com/google/uuid"
)

type CreditRole string

const (
	RoleHost     CreditRole = "host"     // typically show-level
	RoleGuest    CreditRole = "guest"    // typically episode-level
	RoleDirector CreditRole = "director" // documentaries
	RoleProducer CreditRole = "producer"
)

func (r CreditRole) IsValid() bool {
	switch r {
	case RoleHost, RoleGuest, RoleDirector, RoleProducer:
		return true
	}
	return false
}

type Person struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Bio       string    `json:"bio,omitempty"`
	ImageURL  string    `json:"image_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Credit struct {
	Person    Person     `json:"person"`
	Role      CreditRole `json:"role"`
	SortOrder int        `json:"sort_order"`
}
