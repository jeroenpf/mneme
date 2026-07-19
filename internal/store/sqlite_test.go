package store

import (
	"context"
	"path/filepath"
	"testing"
)

// TestNewDispatchesSQLite proves store.New builds a working SQLiteStore from a
// sqlite DSN (modernc opens the file, pragmas apply) and that Ping/Close work —
// the end-to-end boot check for the self-contained binary (plan p1-t3/t4).
func TestNewDispatchesSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mneme.db")
	st, err := New(context.Background(), "sqlite:"+dbPath)
	if err != nil {
		t.Fatalf("New(sqlite): %v", err)
	}
	defer st.Close()

	if _, ok := st.(*SQLiteStore); !ok {
		t.Fatalf("New returned %T, want *SQLiteStore", st)
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
