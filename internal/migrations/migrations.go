package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite" // pure-Go modernc driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	dsnpkg "github.com/jeroenpf/mneme/internal/dsn"
)

//go:embed sql/postgres/*.sql sql/sqlite/*.sql
var FS embed.FS

// Up applies all pending migrations against dsn using the embedded SQL files,
// dispatching on the DSN scheme: postgres:// uses the pgx5 driver and the
// sql/postgres set; sqlite: / file: / *.db uses the pure-Go sqlite driver and
// the sql/sqlite set. Safe to call on every startup; no-ops when the database
// is already at head.
func Up(dsn string) error {
	if dsnpkg.IsSQLite(dsn) {
		return up("sql/sqlite", "sqlite://"+dsnpkg.SQLiteFilePath(dsn))
	}
	return up("sql/postgres", "pgx5://"+stripScheme(dsn))
}

// up runs the migration set under subdir against the driver URL, treating
// "already at head" as success.
func up(subdir, dbURL string) error {
	sub, err := fs.Sub(FS, subdir)
	if err != nil {
		return fmt.Errorf("sub fs %q: %w", subdir, err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
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
