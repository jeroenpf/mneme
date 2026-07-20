package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver

	"github.com/jeroenpf/mneme/internal/dsn"
)

// SQLiteStore implements the full Store contract.
var _ Store = (*SQLiteStore)(nil)

// SQLiteStore is the pure-Go Store implementation backing the self-contained
// binary. It wraps a database/sql handle over modernc.org/sqlite; there is no
// cgo and no external database. Postgres remains the maintainer's primary
// backend — SQLite is the distribution artifact for beta testers.
type SQLiteStore struct {
	db *sql.DB
	// vectorMaxDist mirrors PostgresStore.vectorMaxDist: the hybrid-search
	// relevance floor (cosine distance). Set once at startup.
	vectorMaxDist float64
}

// sqlitePragmas are applied per-connection via the modernc DSN. WAL keeps
// readers concurrent with the single writer; busy_timeout lets a briefly-locked
// write wait rather than fail; foreign_keys enforces our FK constraints (off by
// default in SQLite).
const sqlitePragmas = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

// NewSQLiteStore opens (creating on first run) the SQLite database named by the
// DSN and verifies it with a ping. The parent directory is created if missing
// so a fresh install boots against a path that does not yet exist. store.New
// dispatches here for sqlite: / file: / *.db DSNs.
func NewSQLiteStore(ctx context.Context, connDSN string) (*SQLiteStore, error) {
	path := dsn.SQLiteFilePath(connDSN)
	if path == "" {
		return nil, fmt.Errorf("sqlite dsn %q has no file path", connDSN)
	}
	// First-run creation: ensure the containing directory exists for a real
	// file path (skip the in-memory / named forms).
	if path != ":memory:" && path[0] != ':' {
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite dir: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", path+"?"+sqlitePragmas)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

// SetSearchMaxDist sets the hybrid-search relevance floor (cosine distance).
func (s *SQLiteStore) SetSearchMaxDist(d float64) { s.vectorMaxDist = d }

// Ping verifies the underlying connection is alive — used by /health.
func (s *SQLiteStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Close releases the database handle.
func (s *SQLiteStore) Close() { s.db.Close() }
