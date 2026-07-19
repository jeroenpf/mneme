package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	dsnpkg "github.com/jeroenpfeil/mneme/internal/dsn"
)

//go:embed sql/*.sql
var FS embed.FS

// Up applies all pending migrations against dsn using the embedded SQL files,
// dispatching on the DSN scheme (postgres vs sqlite). Safe to call on every
// startup; no-ops when the database is already at head.
func Up(dsn string) error {
	if dsnpkg.IsSQLite(dsn) {
		return upSQLite(dsn)
	}
	return upPostgres(dsn)
}

// upSQLite applies the SQLite migration set. In P1 the set is empty (the SQLite
// schema is authored in P2's migrations split), so this is a no-op that still
// accepts the DSN so the self-contained binary boots against a fresh file.
func upSQLite(dsn string) error {
	return nil
}

func upPostgres(dsn string) error {
	sub, err := fs.Sub(FS, "sql")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripScheme(dsn))
	if err != nil {
		return fmt.Errorf("new migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// stripScheme converts "postgres://..." or "postgresql://..." to the
// bare "user:pass@host/db?..." form expected after the migrate driver
// prefix "pgx5://".
func stripScheme(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
			return dsn[len(prefix):]
		}
	}
	return dsn
}
