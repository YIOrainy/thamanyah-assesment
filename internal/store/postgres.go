package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/yazeedalorainy/thmanyah/internal/auth"
	"github.com/yazeedalorainy/thmanyah/internal/catalog"
)

// Open returns a Bun DB backed by Postgres with the given pool settings.
func Open(dsn string, maxOpenConns, maxIdleConns int) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	return bun.NewDB(sqldb, pgdialect.New()), nil
}

func isUniqueViolation(err error) bool {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) {
		return pgErr.Field('C') == "23505"
	}
	return false
}

type showRow struct {
	bun.BaseModel `bun:"table:shows,alias:s"`

	ID          uuid.UUID `bun:"id,pk"`
	Title       string    `bun:"title"`
	Slug        string    `bun:"slug"`
	Description string    `bun:"description"`
	Format      string    `bun:"format"`
	Language    string    `bun:"language"`
	Status      string    `bun:"status"`
	CreatedBy   uuid.UUID `bun:"created_by"`
	UpdatedBy   uuid.UUID `bun:"updated_by"`
	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
}

func toShowRow(s *catalog.Show) showRow {
	return showRow{
		ID: s.ID, Title: s.Title, Slug: s.Slug, Description: s.Description,
		Format: string(s.Format), Language: s.Language, Status: string(s.Status),
		CreatedBy: s.CreatedBy, UpdatedBy: s.UpdatedBy,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func (row showRow) toDomain() *catalog.Show {
	return &catalog.Show{
		ID: row.ID, Title: row.Title, Slug: row.Slug, Description: row.Description,
		Format: catalog.Format(row.Format), Language: row.Language, Status: catalog.Status(row.Status),
		CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type PostgresShowRepository struct {
	db *bun.DB
}

func NewPostgresShowRepository(db *bun.DB) *PostgresShowRepository {
	return &PostgresShowRepository{db: db}
}

var _ ShowRepository = (*PostgresShowRepository)(nil)

func (r *PostgresShowRepository) Create(ctx context.Context, show *catalog.Show) error {
	row := toShowRow(show)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *PostgresShowRepository) Update(ctx context.Context, show *catalog.Show) error {
	show.UpdatedAt = time.Now().UTC()
	row := toShowRow(show)
	res, err := r.db.NewUpdate().Model(&row).WherePK().Exec(ctx)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresShowRepository) GetByID(ctx context.Context, id uuid.UUID) (*catalog.Show, error) {
	var row showRow
	err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *PostgresShowRepository) GetBySlug(ctx context.Context, slug string) (*catalog.Show, error) {
	var row showRow
	err := r.db.NewSelect().Model(&row).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func applyShowFilters(q *bun.SelectQuery, f ShowFilter) *bun.SelectQuery {
	if f.Format != "" {
		q = q.Where("format = ?", string(f.Format))
	}
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.Language != "" {
		q = q.Where("language = ?", f.Language)
	}
	return q
}

func (r *PostgresShowRepository) List(ctx context.Context, f ShowFilter) ([]*catalog.Show, error) {
	var rows []showRow
	q := applyShowFilters(r.db.NewSelect().Model(&rows), f)
	if f.Cursor != "" {
		ts, id, err := DecodeCursor(f.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(created_at, id) < (?, ?)", ts, id) // keyset for DESC order
	} else if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	q = q.OrderExpr("created_at DESC, id DESC")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*catalog.Show, len(rows))
	for i := range rows {
		out[i] = rows[i].toDomain()
	}
	return out, nil
}

func (r *PostgresShowRepository) Count(ctx context.Context, f ShowFilter) (int, error) {
	return applyShowFilters(r.db.NewSelect().Model((*showRow)(nil)), f).Count(ctx)
}

type episodeRow struct {
	bun.BaseModel `bun:"table:episodes,alias:e"`

	ID              uuid.UUID  `bun:"id,pk"`
	ShowID          uuid.UUID  `bun:"show_id"`
	Title           string     `bun:"title"`
	Slug            string     `bun:"slug"`
	Description     string     `bun:"description"`
	EpisodeNumber   int        `bun:"episode_number"`
	ContentType     string     `bun:"content_type"`
	Language        string     `bun:"language"`
	DurationSeconds int        `bun:"duration_seconds"`
	Status          string     `bun:"status"`
	PublishedAt     *time.Time `bun:"published_at"`
	SearchText      string     `bun:"search_text"`
	CreatedBy       uuid.UUID  `bun:"created_by"`
	UpdatedBy       uuid.UUID  `bun:"updated_by"`
	CreatedAt       time.Time  `bun:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at"`
}

func toEpisodeRow(e *catalog.Episode) episodeRow {
	return episodeRow{
		ID: e.ID, ShowID: e.ShowID, Title: e.Title, Slug: e.Slug, Description: e.Description,
		EpisodeNumber: e.EpisodeNumber, ContentType: string(e.ContentType), Language: e.Language,
		DurationSeconds: e.DurationSeconds, Status: string(e.Status), PublishedAt: e.PublishedAt,
		SearchText: e.SearchText, CreatedBy: e.CreatedBy, UpdatedBy: e.UpdatedBy,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

func (row episodeRow) toDomain() *catalog.Episode {
	return &catalog.Episode{
		ID: row.ID, ShowID: row.ShowID, Title: row.Title, Slug: row.Slug, Description: row.Description,
		EpisodeNumber: row.EpisodeNumber, ContentType: catalog.ContentType(row.ContentType), Language: row.Language,
		DurationSeconds: row.DurationSeconds, Status: catalog.Status(row.Status), PublishedAt: row.PublishedAt,
		SearchText: row.SearchText, CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type PostgresEpisodeRepository struct {
	db *bun.DB
}

func NewPostgresEpisodeRepository(db *bun.DB) *PostgresEpisodeRepository {
	return &PostgresEpisodeRepository{db: db}
}

var (
	_ EpisodeRepository = (*PostgresEpisodeRepository)(nil)
	_ Searcher          = (*PostgresEpisodeRepository)(nil)
)

// SearchEpisodes runs a Postgres full-text query over published episodes.
func (r *PostgresEpisodeRepository) SearchEpisodes(ctx context.Context, query string, f SearchFilter) ([]*catalog.Episode, error) {
	var rows []episodeRow
	q := r.db.NewSelect().Model(&rows).Where("status = ?", string(catalog.StatusPublished))
	if query != "" {
		q = q.Where("search_tsv @@ websearch_to_tsquery('simple', ?)", query)
	}
	if f.Language != "" {
		q = q.Where("language = ?", f.Language)
	}
	if f.Cursor != "" {
		ts, id, err := DecodeCursor(f.Cursor)
		if err != nil {
			return nil, err
		}
		q = q.Where("(published_at, id) < (?, ?)", ts, id) // keyset (all results are published)
	}
	q = q.OrderExpr("published_at DESC NULLS LAST, id DESC")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*catalog.Episode, len(rows))
	for i := range rows {
		out[i] = rows[i].toDomain()
	}
	return out, nil
}

func (r *PostgresEpisodeRepository) Create(ctx context.Context, ep *catalog.Episode) error {
	row := toEpisodeRow(ep)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *PostgresEpisodeRepository) Update(ctx context.Context, ep *catalog.Episode) error {
	ep.UpdatedAt = time.Now().UTC()
	row := toEpisodeRow(ep)
	res, err := r.db.NewUpdate().Model(&row).WherePK().Exec(ctx)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresEpisodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*catalog.Episode, error) {
	var row episodeRow
	err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *PostgresEpisodeRepository) GetBySlug(ctx context.Context, slug string) (*catalog.Episode, error) {
	var row episodeRow
	err := r.db.NewSelect().Model(&row).Where("slug = ?", slug).Limit(1).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func applyEpisodeFilters(q *bun.SelectQuery, f EpisodeFilter) *bun.SelectQuery {
	if f.ShowID != uuid.Nil {
		q = q.Where("show_id = ?", f.ShowID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	return q
}

func (r *PostgresEpisodeRepository) Count(ctx context.Context, f EpisodeFilter) (int, error) {
	return applyEpisodeFilters(r.db.NewSelect().Model((*episodeRow)(nil)), f).Count(ctx)
}

func (r *PostgresEpisodeRepository) List(ctx context.Context, f EpisodeFilter) ([]*catalog.Episode, error) {
	var rows []episodeRow
	q := applyEpisodeFilters(r.db.NewSelect().Model(&rows), f)
	q = q.OrderExpr("episode_number ASC")
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*catalog.Episode, len(rows))
	for i := range rows {
		out[i] = rows[i].toDomain()
	}
	return out, nil
}

// --- users ---

type userRow struct {
	bun.BaseModel `bun:"table:cms_users,alias:u"`

	ID           uuid.UUID `bun:"id,pk"`
	Name         string    `bun:"name"`
	Email        string    `bun:"email"`
	PasswordHash string    `bun:"password_hash"`
	Role         string    `bun:"role"`
	CreatedAt    time.Time `bun:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at"`
}

func toUserRow(u *auth.User) userRow {
	return userRow{
		ID: u.ID, Name: u.Name, Email: u.Email, PasswordHash: u.PasswordHash,
		Role: string(u.Role), CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func (row userRow) toDomain() *auth.User {
	return &auth.User{
		ID: row.ID, Name: row.Name, Email: row.Email, PasswordHash: row.PasswordHash,
		Role: auth.Role(row.Role), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type PostgresUserRepository struct {
	db *bun.DB
}

func NewPostgresUserRepository(db *bun.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

var _ UserRepository = (*PostgresUserRepository)(nil)

func (r *PostgresUserRepository) Create(ctx context.Context, u *auth.User) error {
	row := toUserRow(u)
	if _, err := r.db.NewInsert().Model(&row).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	var row userRow
	err := r.db.NewSelect().Model(&row).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	var row userRow
	err := r.db.NewSelect().Model(&row).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}
