package dsn

import "testing"

func TestIsSQLite(t *testing.T) {
	cases := []struct {
		dsn  string
		want bool
	}{
		// Postgres forms — the default backend.
		{"postgres://mneme:mneme@localhost:5432/mneme?sslmode=disable", false},
		{"postgresql://x@y/z", false},
		// SQLite scheme forms.
		{"sqlite:///tmp/mneme.db", true},
		{"sqlite:mneme.db", true},
		{"sqlite3:///var/data.db", true},
		{"file:/tmp/x.db", true},
		{"file:test.db?_pragma=busy_timeout(5000)", true},
		// Bare paths that name a sqlite file.
		{"/tmp/mneme.db", true},
		{"mneme.db", true},
		{"./data/store.sqlite", true},
		{":memory:", true},
		// A bare path that is not obviously sqlite defaults to postgres.
		{"localhost/mneme", false},
	}
	for _, c := range cases {
		if got := IsSQLite(c.dsn); got != c.want {
			t.Errorf("IsSQLite(%q) = %v, want %v", c.dsn, got, c.want)
		}
	}
}

func TestSQLiteFilePath(t *testing.T) {
	cases := []struct {
		dsn  string
		want string
	}{
		{"sqlite:///tmp/mneme.db", "/tmp/mneme.db"},
		{"sqlite:mneme.db", "mneme.db"},
		{"sqlite3:///var/data.db", "/var/data.db"},
		{"file:/tmp/x.db", "/tmp/x.db"},
		{"file:///tmp/x.db", "/tmp/x.db"},
		{"file:test.db?_pragma=busy_timeout(5000)", "test.db"},
		{"/tmp/mneme.db", "/tmp/mneme.db"},
		{"mneme.db?_pragma=journal_mode(WAL)", "mneme.db"},
		{":memory:", ":memory:"},
	}
	for _, c := range cases {
		if got := SQLiteFilePath(c.dsn); got != c.want {
			t.Errorf("SQLiteFilePath(%q) = %q, want %q", c.dsn, got, c.want)
		}
	}
}
