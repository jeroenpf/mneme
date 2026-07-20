package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"

	"github.com/jeroenpf/mneme/internal/migrations"
)

// TestSQLiteMigrationsApplyToFreshDB runs the SQLite migration set against a
// fresh temp file and asserts the full schema materialized: every base table,
// one FTS5 virtual table per searchable type, and the updated_at + FTS-sync
// triggers (plan p2-t6). This is the schema regression guard for the SQLite
// backend.
func TestSQLiteMigrationsApplyToFreshDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up("sqlite:" + dbPath); err != nil {
		t.Fatalf("Up(sqlite): %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	wantObjects := map[string]string{
		// base tables
		"projects":        "table",
		"documents":       "table",
		"decisions":       "table",
		"snippets":        "table",
		"journal_entries": "table",
		"solutions":       "table",
		"memories":        "table",
		"env_entries":     "table",
		"embeddings":      "table",
		"embed_failures":  "table",
		// FTS5 virtual tables (one per searchable type)
		"documents_fts": "table",
		"decisions_fts": "table",
		"snippets_fts":  "table",
		"solutions_fts": "table",
		"journal_fts":   "table",
		"memories_fts":  "table",
		// updated_at triggers (mirror the Postgres set_updated_at triggers)
		"documents_set_updated_at":       "trigger",
		"decisions_set_updated_at":       "trigger",
		"snippets_set_updated_at":        "trigger",
		"journal_entries_set_updated_at": "trigger",
		"solutions_set_updated_at":       "trigger",
		// representative FTS sync triggers
		"documents_fts_ai": "trigger",
		"documents_fts_ad": "trigger",
		"documents_fts_au": "trigger",
	}
	for name, typ := range wantObjects {
		var got string
		err := db.QueryRow(
			`SELECT type FROM sqlite_master WHERE name = ?`, name).Scan(&got)
		if err == sql.ErrNoRows {
			t.Errorf("schema object %q (%s) missing", name, typ)
			continue
		}
		if err != nil {
			t.Fatalf("query sqlite_master for %q: %v", name, err)
		}
		if got != typ {
			t.Errorf("object %q: got type %q, want %q", name, got, typ)
		}
	}
}

// TestUpAcceptsSQLiteDSN checks migrations.Up dispatches on the DSN scheme and
// does not choke on a sqlite: DSN. In P1 the SQLite migration set is empty, so
// Up is a no-op that must still return nil (the self-contained binary boots
// against a fresh file). P2 fills sql/sqlite/ and this same call builds the
// schema.
func TestUpAcceptsSQLiteDSN(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up("sqlite:" + dbPath); err != nil {
		t.Fatalf("Up(sqlite): %v", err)
	}
}

// TestPublicIDsMigrationRollsBackAndForward exercises migration 015's down
// and up files: after a full Up the public_id column exists; migrating down
// to 014 drops it; migrating back up to 015 restores it. Targets version 015
// explicitly so it stays correct as later migrations are added.
func TestPublicIDsMigrationRollsBackAndForward(t *testing.T) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("mneme"),
		tcpostgres.WithUsername("mneme"),
		tcpostgres.WithPassword("mneme"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	sub, err := fs.Sub(migrations.FS, "sql/postgres")
	if err != nil {
		t.Fatalf("sub fs: %v", err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		t.Fatalf("iofs: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripScheme(dsn))
	if err != nil {
		t.Fatalf("new migrate: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("up: %v", err)
	}
	if !columnExists(t, dsn, "documents", "public_id") {
		t.Fatal("after up: documents.public_id should exist")
	}

	// Roll back to 014 (undoing every migration after it, incl. 015), so
	// public_id is dropped regardless of how many migrations follow 015.
	if err := m.Migrate(14); err != nil {
		t.Fatalf("migrate down to 014: %v", err)
	}
	if columnExists(t, dsn, "documents", "public_id") {
		t.Fatal("after rollback: documents.public_id should be gone")
	}

	if err := m.Migrate(15); err != nil {
		t.Fatalf("migrate up to 015: %v", err)
	}
	if !columnExists(t, dsn, "solutions", "public_id") {
		t.Fatal("after re-apply: solutions.public_id should exist again")
	}
}

func columnExists(t *testing.T, dsn, table, column string) bool {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM information_schema.columns
		   WHERE table_name = $1 AND column_name = $2
		 )`, table, column).Scan(&exists); err != nil {
		t.Fatalf("check column: %v", err)
	}
	return exists
}

// stripScheme mirrors the unexported helper in the migrations package: the
// pgx5 driver wants the dsn without the postgres:// scheme prefix.
func stripScheme(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
			return dsn[len(prefix):]
		}
	}
	return dsn
}

var _ = fmt.Sprintf // keep fmt imported for future diagnostics
