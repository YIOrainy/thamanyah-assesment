package store

import (
	"context"
	"errors"
	"log/slog"

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
		slog.Info("store closed")
	}
}

// New builds the repositories for the configured backend (Postgres or memory).
func New(cfg config.StoreConfig) (*Repositories, error) {
	if cfg.Postgres.Enabled {
		slog.Info("store backend selected",
			"backend", "postgres",
			"max_open_conns", cfg.Postgres.MaxOpenConns,
			"max_idle_conns", cfg.Postgres.MaxIdleConns,
		)
		db, err := Open(cfg.Postgres.DSN, cfg.Postgres.MaxOpenConns, cfg.Postgres.MaxIdleConns)
		if err != nil {
			slog.Error("open postgres store failed", "error", err)
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

	slog.Info("store backend selected", "backend", "memory", "actors", cfg.Memory.Actors)
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
		slog.InfoContext(ctx, "bootstrap admin skipped", "reason", "email_not_configured")
		return nil
	}
	if _, err := r.Users.GetByEmail(ctx, ba.Email); err == nil {
		slog.InfoContext(ctx, "bootstrap admin already exists", "email", ba.Email)
		return nil // already exists
	} else if !errors.Is(err, ErrNotFound) {
		slog.ErrorContext(ctx, "bootstrap admin lookup failed", "email", ba.Email, "error", err)
		return err
	}
	hash, err := auth.HashPassword(ba.Password)
	if err != nil {
		slog.ErrorContext(ctx, "bootstrap admin password hash failed", "email", ba.Email, "error", err)
		return err
	}
	user := auth.NewUser("Admin", ba.Email, hash, auth.RoleAdmin)
	if err := r.Users.Create(ctx, user); err != nil {
		slog.ErrorContext(ctx, "bootstrap admin create failed", "email", ba.Email, "error", err)
		return err
	}
	slog.InfoContext(ctx, "bootstrap admin created", "email", ba.Email, "user_id", user.ID)
	return nil
}
