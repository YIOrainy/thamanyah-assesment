//go:build integration

package store

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

// newPostgresDB starts a throwaway Postgres container, applies the catalog
// migration, and returns a Bun DB. The container is torn down on cleanup.
func newPostgresDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("thmanyah"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := Open(dsn, 10, 10)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	applyMigrations(t, db)
	return db
}

// applyMigrations runs every *.up.sql in order against the database.
func applyMigrations(t *testing.T, db *bun.DB) {
	t.Helper()
	files, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := db.ExecContext(context.Background(), string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", f, err)
		}
	}
}

func ensurePostgresShow(db *bun.DB) ensureShowFunc {
	return func(t *testing.T, showID uuid.UUID) {
		row := showRow{
			ID: showID, Title: "parent", Slug: "show-" + showID.String(),
			Format: "podcast", Language: "ar", Status: "draft",
			CreatedBy: showID, UpdatedBy: showID,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		if _, err := db.NewInsert().Model(&row).On("CONFLICT (id) DO NOTHING").Exec(context.Background()); err != nil {
			t.Fatalf("ensure show: %v", err)
		}
	}
}

func truncate(t *testing.T, db *bun.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), "TRUNCATE episodes, shows, cms_users, external_refs CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestPostgresShowRepository(t *testing.T) {
	db := newPostgresDB(t)
	runShowRepositoryContract(t, func(t *testing.T) ShowRepository {
		truncate(t, db) // fresh slate per subtest, matching the memory adapter's isolation
		return NewPostgresShowRepository(db)
	})
}

func TestPostgresEpisodeRepository(t *testing.T) {
	db := newPostgresDB(t)
	runEpisodeRepositoryContract(t, func(t *testing.T) EpisodeRepository {
		truncate(t, db)
		return NewPostgresEpisodeRepository(db)
	}, ensurePostgresShow(db))
}

func TestPostgresSearcher(t *testing.T) {
	db := newPostgresDB(t)
	runSearcherContract(t, func(t *testing.T) searchSetup {
		truncate(t, db)
		r := NewPostgresEpisodeRepository(db)
		return searchSetup{repo: r, search: r, ensureShow: ensurePostgresShow(db)}
	})
}

func TestPostgresUserRepository(t *testing.T) {
	db := newPostgresDB(t)
	runUserRepositoryContract(t, func(t *testing.T) UserRepository {
		truncate(t, db)
		return NewPostgresUserRepository(db)
	})
}

func TestPostgresExternalRefRepository(t *testing.T) {
	db := newPostgresDB(t)
	runExternalRefContract(t, func(t *testing.T) ExternalRefRepository {
		truncate(t, db)
		return NewPostgresExternalRefRepository(db)
	})
}
