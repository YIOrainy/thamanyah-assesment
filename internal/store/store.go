package store

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

var (
	ErrNotFound = errors.New("store: not found")
	ErrConflict = errors.New("store: already exists")
)

type ShowFilter struct {
	Format   catalog.Format
	Status   catalog.Status
	Language string
	Limit    int
	Offset   int    // cms
	Cursor   string // discovery
}

type EpisodeFilter struct {
	ShowID uuid.UUID
	Status catalog.Status
	Limit  int
	Offset int
}

type ShowRepository interface {
	Create(ctx context.Context, show *catalog.Show) error
	Update(ctx context.Context, show *catalog.Show) error
	GetByID(ctx context.Context, id uuid.UUID) (*catalog.Show, error)
	GetBySlug(ctx context.Context, slug string) (*catalog.Show, error)
	List(ctx context.Context, f ShowFilter) ([]*catalog.Show, error)
	Count(ctx context.Context, f ShowFilter) (int, error)
}

type EpisodeRepository interface {
	Create(ctx context.Context, ep *catalog.Episode) error
	Update(ctx context.Context, ep *catalog.Episode) error
	GetByID(ctx context.Context, id uuid.UUID) (*catalog.Episode, error)
	GetBySlug(ctx context.Context, slug string) (*catalog.Episode, error)
	List(ctx context.Context, f EpisodeFilter) ([]*catalog.Episode, error)
	Count(ctx context.Context, f EpisodeFilter) (int, error)
}

type UserRepository interface {
	Create(ctx context.Context, u *auth.User) error
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
}
