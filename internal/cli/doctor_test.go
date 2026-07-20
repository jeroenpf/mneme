package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/store"
)

// migratedStore opens a fresh, migrated SQLite store for check tests.
func migratedStore(t *testing.T) store.Store {
	t.Helper()
	dsn := "sqlite://" + filepath.Join(t.TempDir(), "doctor.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestCheckConfig(t *testing.T) {
	if got := checkConfig(nil, errors.New("bad dsn")); got.level != levelFail {
		t.Errorf("load error should fail, got %v", got.level)
	}
	cfg := &config.Config{DSN: "sqlite:///x.db", Host: "127.0.0.1", Port: "8765"}
	if got := checkConfig(cfg, nil); got.level != levelOK {
		t.Errorf("valid config should pass, got %v (%s)", got.level, got.detail)
	}
}

func TestCheckTLS(t *testing.T) {
	// Plain HTTP (localhost) — nothing to check.
	if got := checkTLS(&config.Config{Host: "127.0.0.1", Port: "8765"}); got.level != levelOK {
		t.Errorf("localhost mode should be OK, got %v", got.level)
	}

	// HTTPS configured but cert files missing → fail.
	miss := &config.Config{TLSCert: "/no/such/cert.pem", TLSKey: "/no/such/key.pem"}
	if got := checkTLS(miss); got.level != levelFail {
		t.Errorf("missing certs should fail, got %v", got.level)
	}

	// HTTPS with real cert files present.
	dir := t.TempDir()
	cert := filepath.Join(dir, "c.pem")
	key := filepath.Join(dir, "k.pem")
	for _, p := range []string{cert, key} {
		if err := os.WriteFile(p, []byte("pem"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tlsCfg := &config.Config{TLSCert: cert, TLSKey: key}

	// hosts entry present → OK.
	oldHosts := hostsPath
	t.Cleanup(func() { hostsPath = oldHosts })
	present := filepath.Join(dir, "hosts_ok")
	os.WriteFile(present, []byte("127.0.0.1\tmneme.dev\n"), 0o644)
	hostsPath = present
	if got := checkTLS(tlsCfg); got.level != levelOK {
		t.Errorf("certs + hosts present should be OK, got %v (%s)", got.level, got.detail)
	}

	// hosts entry missing → warn (cert fine, but mneme.dev won't resolve).
	absent := filepath.Join(dir, "hosts_missing")
	os.WriteFile(absent, []byte("127.0.0.1 localhost\n"), 0o644)
	hostsPath = absent
	if got := checkTLS(tlsCfg); got.level != levelWarn {
		t.Errorf("certs present but hosts missing should warn, got %v (%s)", got.level, got.detail)
	}
}

func TestCheckDatabaseAndSearch(t *testing.T) {
	st := migratedStore(t)
	ctx := context.Background()

	if got := checkDatabase(ctx, st); got.level != levelOK {
		t.Errorf("migrated store should pass DB check, got %v (%s)", got.level, got.detail)
	}
	if got := checkSearch(ctx, st); got.level != levelOK {
		t.Errorf("search on migrated store should pass, got %v (%s)", got.level, got.detail)
	}
}

func TestCheckEmbeddings(t *testing.T) {
	st := migratedStore(t)
	ctx := context.Background()

	// No key → lexical-only (a warn-level "heads up", not a failure).
	lexical := checkEmbeddings(ctx, &config.Config{}, st)
	if lexical.level == levelFail {
		t.Errorf("lexical-only must not fail; got %v", lexical.level)
	}
	if lexical.level != levelWarn {
		t.Errorf("no key should warn (embeddings off), got %v", lexical.level)
	}

	// Key configured → OK, reporting the provider/model.
	withKey := checkEmbeddings(ctx, &config.Config{VoyageKey: "pa-x", VoyageModel: "voyage-4-large"}, st)
	if withKey.level != levelOK {
		t.Errorf("configured embeddings should be OK, got %v (%s)", withKey.level, withKey.detail)
	}
}
