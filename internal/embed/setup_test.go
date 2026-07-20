package embed_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/store"
)

// embedTestDSN is the container DSN shared across the embed_test package;
// each test opens its own pool off it and TRUNCATEs for a clean slate.
// Mirrors internal/store/postgres_test.go's TestMain + newStore harness so
// the worker test is container-backed and self-contained.
var embedTestDSN string

func TestMain(m *testing.M) {
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
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection string: %v\n", err)
		os.Exit(1)
	}
	embedTestDSN = dsn

	if err := migrations.Up(embedTestDSN); err != nil {
		fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// newEmbedStore returns a fresh-state store, TRUNCATEing so each test
// starts clean. Accepts testing.TB so benchmarks can use it too.
func newEmbedStore(t testing.TB) *store.PostgresStore {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, embedTestDSN)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx,
		`TRUNCATE documents, projects, embeddings, embed_failures RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return store.NewWithPool(pool)
}

func ptrs(s string) *string { return &s }

// seedProject inserts a project slug so FK-constrained rows can reference it.
func seedProject(t testing.TB, s *store.PostgresStore, slug string) {
	t.Helper()
	if _, err := s.Pool().Exec(context.Background(),
		`INSERT INTO projects (name, slug) VALUES ($1, $1)`, slug); err != nil {
		t.Fatalf("seed project %q: %v", slug, err)
	}
}
