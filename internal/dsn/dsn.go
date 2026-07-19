// Package dsn is the single source of truth for storage-backend selection from
// a connection string. Both store.New and migrations.Up dispatch on the DSN
// scheme, so the detection lives here to keep the two in lockstep.
package dsn

import "strings"

// sqliteSchemes are the prefixes that unambiguously name the SQLite backend.
// "file:" is included because the modernc driver's own URI form uses it.
var sqliteSchemes = []string{"sqlite://", "sqlite3://", "file://", "sqlite:", "sqlite3:", "file:"}

// sqliteFileSuffixes let a bare path (no scheme) name a SQLite database.
var sqliteFileSuffixes = []string{".db", ".sqlite", ".sqlite3"}

// IsSQLite reports whether dsn selects the SQLite backend. Postgres is the
// default: postgres:// / postgresql:// and any string not recognised as SQLite
// route to PostgresStore.
func IsSQLite(dsn string) bool {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return false
	}
	for _, s := range sqliteSchemes {
		if strings.HasPrefix(dsn, s) {
			return true
		}
	}
	path := dsn
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == ":memory:" {
		return true
	}
	for _, suf := range sqliteFileSuffixes {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}

// SQLiteFilePath extracts the on-disk file path from a SQLite DSN, stripping any
// scheme prefix and query string. "sqlite:///abs/x.db" → "/abs/x.db",
// "file:rel.db?_pragma=..." → "rel.db", ":memory:" → ":memory:". The result is
// what modernc.org/sqlite expects as its database name (before pragmas).
func SQLiteFilePath(dsn string) string {
	s := dsn
	for _, prefix := range sqliteSchemes {
		if strings.HasPrefix(s, prefix) {
			s = s[len(prefix):]
			break
		}
	}
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}
	return s
}
