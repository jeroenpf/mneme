package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jeroenpfeil/mneme/internal/backup"
	"github.com/jeroenpfeil/mneme/internal/config"
	"github.com/jeroenpfeil/mneme/internal/dsn"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// checkLevel is a diagnostic outcome. Only levelFail makes `mneme doctor` exit
// non-zero; levelWarn is advisory (yellow), levelOK is green.
type checkLevel int

const (
	levelOK checkLevel = iota
	levelWarn
	levelFail
)

// check is one line of the doctor scorecard.
type check struct {
	name   string
	level  checkLevel
	detail string
}

func ok(name, detail string) check   { return check{name, levelOK, detail} }
func warn(name, detail string) check { return check{name, levelWarn, detail} }
func fail(name, detail string) check { return check{name, levelFail, detail} }

// checkConfig reports whether configuration resolved cleanly and which storage
// backend it selects.
func checkConfig(cfg *config.Config, loadErr error) check {
	if loadErr != nil {
		return fail("config", loadErr.Error())
	}
	backend := "PostgreSQL"
	if dsn.IsSQLite(cfg.DSN) {
		backend = "SQLite"
	}
	return ok("config", fmt.Sprintf("%s backend, listening on %s", backend, cfg.ListenAddr()))
}

// checkDatabase verifies the store is reachable and its schema is present by
// exercising a query that touches a migrated table.
func checkDatabase(ctx context.Context, st store.Store) check {
	if err := st.Ping(ctx); err != nil {
		return fail("database", "not reachable: "+err.Error())
	}
	if _, err := st.ListProjects(ctx); err != nil {
		return fail("database", "schema not ready (migrations pending?): "+err.Error())
	}
	return ok("database", "reachable, schema at head")
}

// checkSearch runs a trivial query to confirm the full-text search path works
// end-to-end (FTS index + fusion), independent of whether embeddings are on.
func checkSearch(ctx context.Context, st store.Store) check {
	if _, err := st.Search(ctx, "diagnostic probe", store.SearchFilter{}); err != nil {
		return fail("search", "query failed: "+err.Error())
	}
	return ok("search", "full-text search responding")
}

// checkEmbeddings reports whether semantic search is configured. No key is not a
// failure — it is the fully-local lexical-only mode — but it is worth flagging.
func checkEmbeddings(ctx context.Context, cfg *config.Config, st store.Store) check {
	if cfg.VoyageKey == "" {
		return warn("embeddings", "disabled — lexical-only (FTS) search, nothing leaves your machine")
	}
	model := cfg.VoyageModel
	if model == "" {
		model = "voyage (default)"
	}
	detail := "provider: Voyage, model: " + model
	if st != nil {
		if statuses, err := st.EmbeddingStatus(ctx, cfg.VoyageModel); err == nil {
			var reconciled, missing, failed int
			for _, s := range statuses {
				reconciled += s.Reconciled
				missing += s.Missing
				failed += s.Failed
			}
			detail += fmt.Sprintf(" (%d embedded, %d pending, %d failed)", reconciled, missing, failed)
		}
	}
	return ok("embeddings", detail)
}

// checkBackups confirms the knowledge base is exportable end-to-end — the
// backup path (`mneme export`) actually works against this store — and reports
// how much is protected. It is advisory (a fresh install has nothing to back up
// yet), so an empty store warns rather than fails.
func checkBackups(ctx context.Context, st store.Store) check {
	arch, err := backup.Export(ctx, st)
	if err != nil {
		return fail("backups", "export failed — data is not backup-able: "+err.Error())
	}
	total := len(arch.Documents) + len(arch.Decisions) + len(arch.Snippets) +
		len(arch.Journal) + len(arch.Solutions) + len(arch.Memory) + len(arch.Env)
	if total == 0 {
		return warn("backups", "nothing to back up yet — run `mneme export <file>` once you have knowledge")
	}
	return ok("backups", fmt.Sprintf("%d item(s) exportable via `mneme export <file>`", total))
}

// checkTLS validates the networking posture. Plain-HTTP loopback needs nothing.
// HTTPS mode requires the cert/key to exist (fail if not) and the mneme.dev
// hosts entry to resolve (warn if not — the cert is fine but the name won't).
func checkTLS(cfg *config.Config) check {
	if cfg.TLSCert == "" && cfg.TLSKey == "" {
		return ok("networking", "plain HTTP on loopback ("+cfg.ListenAddr()+") — no certificate needed")
	}
	if !cfg.TLSEnabled() {
		return fail("networking", "HTTPS configured but cert/key file missing: "+cfg.TLSCert)
	}
	if !hostsEntryPresent() {
		return warn("networking", "certificate present, but /etc/hosts is missing the mneme.dev → 127.0.0.1 entry")
	}
	return ok("networking", "HTTPS certificate present and mneme.dev resolves to loopback")
}

// hostsEntryPresent reports whether the system hosts file maps mneme.dev to
// loopback (read-only; tolerant of an unreadable file).
func hostsEntryPresent() bool {
	b, err := os.ReadFile(hostsPath)
	return err == nil && hostsHasMnemeEntry(string(b))
}
