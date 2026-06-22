package store

import (
	"context"
	"errors"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/config"
)

// Repositories is the set of persistence ports, wired to one backend.
type Repositories struct {
	Shows    ShowRepository
	Episodes EpisodeRepository
	Searcher Searcher
	Users    UserRepository
	Refs     ExternalRefRepository
	close    func()
}

func (r *Repositories) Close() {
	if r.close != nil {
		r.close()
	}
}

// New builds the repositories for the configured backend (Postgres or memory).
func New(cfg config.StoreConfig) (*Repositories, error) {
	if cfg.Postgres.Enabled {
		db, err := Open(cfg.Postgres.DSN, cfg.Postgres.MaxOpenConns, cfg.Postgres.MaxIdleConns)
		if err != nil {
			return nil, err
		}
		ep := NewPostgresEpisodeRepository(db)
		return &Repositories{
			Shows:    NewPostgresShowRepository(db),
			Episodes: ep,
			Searcher: ep,
			Users:    NewPostgresUserRepository(db),
			Refs:     NewPostgresExternalRefRepository(db),
			close:    func() { _ = db.Close() },
		}, nil
	}

	sh := NewMemoryShowRepository(cfg.Memory.Actors)
	ep := NewMemoryEpisodeRepository(cfg.Memory.Actors)
	us := NewMemoryUserRepository()
	refs := NewMemoryExternalRefRepository()
	return &Repositories{
		Shows: sh, Episodes: ep, Searcher: ep, Users: us, Refs: refs,
		close: func() { sh.Close(); ep.Close(); us.Close(); refs.Close() },
	}, nil
}

// BootstrapAdmin ensures the configured admin user exists (idempotent).
func (r *Repositories) BootstrapAdmin(ctx context.Context, ba config.BootstrapAdmin) error {
	if ba.Email == "" {
		return nil
	}
	if _, err := r.Users.GetByEmail(ctx, ba.Email); err == nil {
		return nil // already exists
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(ba.Password)
	if err != nil {
		return err
	}
	return r.Users.Create(ctx, auth.NewUser("Admin", ba.Email, hash, auth.RoleAdmin))
}
