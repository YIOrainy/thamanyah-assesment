package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Owner types for external references.
const (
	OwnerShow    = "show"
	OwnerEpisode = "episode"
)

// ExternalRef links a catalog record to its origin in an external source.
// (source, owner_type, external_id) is unique — the idempotency key for imports.
type ExternalRef struct {
	ID          uuid.UUID
	Source      string
	OwnerType   string
	OwnerID     uuid.UUID
	ExternalID  string
	ExternalURL string
	ImportedAt  time.Time
}

// ExternalRefRepository is the persistence port for import provenance.
type ExternalRefRepository interface {
	Create(ctx context.Context, ref *ExternalRef) error
	FindOwner(ctx context.Context, source, ownerType, externalID string) (uuid.UUID, bool, error)
}

func refKey(source, ownerType, externalID string) string {
	return source + "|" + ownerType + "|" + externalID
}

// --- memory ---

type MemoryExternalRefRepository struct {
	*actor
	byKey map[string]ExternalRef
}

func NewMemoryExternalRefRepository() *MemoryExternalRefRepository {
	return &MemoryExternalRefRepository{actor: newActor(), byKey: make(map[string]ExternalRef)}
}

var _ ExternalRefRepository = (*MemoryExternalRefRepository)(nil)

func (r *MemoryExternalRefRepository) Create(ctx context.Context, ref *ExternalRef) error {
	_, err := submit(ctx, r.actor, func() (struct{}, error) {
		k := refKey(ref.Source, ref.OwnerType, ref.ExternalID)
		if _, ok := r.byKey[k]; ok {
			return struct{}{}, ErrConflict
		}
		r.byKey[k] = *ref
		return struct{}{}, nil
	})
	return err
}

func (r *MemoryExternalRefRepository) FindOwner(ctx context.Context, source, ownerType, externalID string) (uuid.UUID, bool, error) {
	type result struct {
		id    uuid.UUID
		found bool
	}
	out, err := submit(ctx, r.actor, func() (result, error) {
		ref, ok := r.byKey[refKey(source, ownerType, externalID)]
		if !ok {
			return result{}, nil
		}
		return result{id: ref.OwnerID, found: true}, nil
	})
	return out.id, out.found, err
}

// --- postgres ---

type externalRefRow struct {
	bun.BaseModel `bun:"table:external_refs,alias:er"`

	ID          uuid.UUID `bun:"id,pk"`
	Source      string    `bun:"source"`
	OwnerType   string    `bun:"owner_type"`
	OwnerID     uuid.UUID `bun:"owner_id"`
	ExternalID  string    `bun:"external_id"`
	ExternalURL string    `bun:"external_url"`
	ImportedAt  time.Time `bun:"imported_at"`
}

type PostgresExternalRefRepository struct {
	db *bun.DB
}

func NewPostgresExternalRefRepository(db *bun.DB) *PostgresExternalRefRepository {
	return &PostgresExternalRefRepository{db: db}
}

var _ ExternalRefRepository = (*PostgresExternalRefRepository)(nil)

func (r *PostgresExternalRefRepository) Create(ctx context.Context, ref *ExternalRef) error {
	row := externalRefRow{
		ID: ref.ID, Source: ref.Source, OwnerType: ref.OwnerType, OwnerID: ref.OwnerID,
		ExternalID: ref.ExternalID, ExternalURL: ref.ExternalURL, ImportedAt: ref.ImportedAt,
	}
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *PostgresExternalRefRepository) FindOwner(ctx context.Context, source, ownerType, externalID string) (uuid.UUID, bool, error) {
	var row externalRefRow
	err := r.db.NewSelect().Model(&row).
		Where("source = ?", source).
		Where("owner_type = ?", ownerType).
		Where("external_id = ?", externalID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, err
	}
	return row.OwnerID, true, nil
}
