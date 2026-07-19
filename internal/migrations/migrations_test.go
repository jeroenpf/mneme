package migrations_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jeroenpfeil/mneme/internal/migrations"
)

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

	sub, err := fs.Sub(migrations.FS, "sql")
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
