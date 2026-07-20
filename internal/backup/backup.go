// Package backup exports and restores all of Mneme's local knowledge as a
// single portable JSON archive (roadmap P6-t7). It is backend-agnostic — it
// reads and writes exclusively through the store.Store interface — so a backup
// taken from Postgres can be restored into a SQLite binary and vice versa.
//
// Restore mints fresh public ids and database uuids for the recreated rows
// (documents keep their slug ids, memory/env keep their natural keys); the
// archive preserves the originals for inspection. Round-trip verification
// therefore compares content, not surrogate identifiers.
package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// FormatVersion is the archive schema version, bumped on incompatible changes.
const FormatVersion = 1

// Archive is the portable snapshot of all local knowledge.
type Archive struct {
	Version   int                    `json:"version"`
	Projects  []*models.Project      `json:"projects"`
	Documents []*models.Document     `json:"documents"`
	Decisions []*models.Decision     `json:"decisions"`
	Snippets  []*models.Snippet      `json:"snippets"`
	Journal   []*models.JournalEntry `json:"journal"`
	Solutions []*models.Solution     `json:"solutions"`
	Memory    []*models.Memory       `json:"memory"`
	Env       []*models.EnvEntry     `json:"env"`
}

// Result counts what an import created versus skipped (already present).
type Result struct {
	Created map[string]int `json:"created"`
	Skipped map[string]int `json:"skipped"`
}

// Export reads every entity through the store into an Archive.
func Export(ctx context.Context, st store.Store) (*Archive, error) {
	a := &Archive{Version: FormatVersion}

	projects, err := st.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("export projects: %w", err)
	}
	for _, p := range projects {
		proj := p.Project
		a.Projects = append(a.Projects, &proj)
	}
	if a.Documents, err = st.ListDocuments(ctx, store.Filter{}); err != nil {
		return nil, fmt.Errorf("export documents: %w", err)
	}
	if a.Decisions, err = st.ListDecisions(ctx, store.DecisionFilter{}); err != nil {
		return nil, fmt.Errorf("export decisions: %w", err)
	}
	if a.Snippets, err = st.ListSnippets(ctx, store.SnippetFilter{}); err != nil {
		return nil, fmt.Errorf("export snippets: %w", err)
	}
	if a.Journal, err = st.ListJournalEntries(ctx, store.JournalFilter{}); err != nil {
		return nil, fmt.Errorf("export journal: %w", err)
	}
	if a.Solutions, err = st.ListSolutions(ctx, store.SolutionFilter{}); err != nil {
		return nil, fmt.Errorf("export solutions: %w", err)
	}
	if a.Memory, err = st.ListMemory(ctx, store.MemoryFilter{}); err != nil {
		return nil, fmt.Errorf("export memory: %w", err)
	}
	// Env is per-project; gather across every project.
	for _, p := range projects {
		entries, err := st.ListEnv(ctx, p.Slug)
		if err != nil {
			return nil, fmt.Errorf("export env for %s: %w", p.Slug, err)
		}
		a.Env = append(a.Env, entries...)
	}
	return a, nil
}

// Write encodes the archive as indented JSON to w.
func (a *Archive) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(a)
}

// Read decodes an archive from r, rejecting an unknown format version.
func Read(r io.Reader) (*Archive, error) {
	var a Archive
	if err := json.NewDecoder(r).Decode(&a); err != nil {
		return nil, fmt.Errorf("decode archive: %w", err)
	}
	if a.Version != FormatVersion {
		return nil, fmt.Errorf("unsupported archive version %d (this build reads version %d)", a.Version, FormatVersion)
	}
	return &a, nil
}

// Import writes every entity in the archive into st. Projects must be restored
// first (documents and scoped memory/env reference them). Rows that already
// exist (duplicate project slug or document id) are skipped so a restore is
// safe to re-run; every other store error aborts.
func Import(ctx context.Context, st store.Store, a *Archive) (*Result, error) {
	res := &Result{Created: map[string]int{}, Skipped: map[string]int{}}

	for _, p := range a.Projects {
		err := st.CreateProject(ctx, &models.Project{Name: p.Name, Slug: p.Slug, Description: p.Description})
		if skipped, e := tally(res, "projects", err, store.ErrDuplicateProject); e != nil {
			return nil, fmt.Errorf("import project %s: %w", p.Slug, e)
		} else if !skipped {
			res.Created["projects"]++
		}
	}
	for _, d := range a.Documents {
		doc := *d
		doc.PublicID = "" // re-minted on create
		err := st.CreateDocument(ctx, &doc)
		if skipped, e := tally(res, "documents", err, store.ErrDuplicateID); e != nil {
			return nil, fmt.Errorf("import document %s: %w", d.ID, e)
		} else if !skipped {
			res.Created["documents"]++
		}
	}
	for _, d := range a.Decisions {
		dec := *d
		if err := st.CreateDecision(ctx, &dec); err != nil {
			return nil, fmt.Errorf("import decision %q: %w", d.Title, err)
		}
		res.Created["decisions"]++
	}
	for _, s := range a.Snippets {
		sn := *s
		if err := st.CreateSnippet(ctx, &sn); err != nil {
			return nil, fmt.Errorf("import snippet %q: %w", s.Title, err)
		}
		res.Created["snippets"]++
	}
	for _, j := range a.Journal {
		je := *j
		if err := st.CreateJournalEntry(ctx, &je); err != nil {
			return nil, fmt.Errorf("import journal %q: %w", j.Summary, err)
		}
		res.Created["journal"]++
	}
	for _, s := range a.Solutions {
		sol := *s
		if err := st.CreateSolution(ctx, &sol); err != nil {
			return nil, fmt.Errorf("import solution: %w", err)
		}
		res.Created["solutions"]++
	}
	for _, m := range a.Memory {
		mem := *m
		if err := st.SetMemory(ctx, &mem); err != nil {
			return nil, fmt.Errorf("import memory %q: %w", m.Key, err)
		}
		res.Created["memory"]++
	}
	for _, e := range a.Env {
		env := *e
		if err := st.SetEnv(ctx, &env); err != nil {
			return nil, fmt.Errorf("import env %q: %w", e.Key, err)
		}
		res.Created["env"]++
	}
	return res, nil
}

// tally classifies a create error: nil is success, a matching duplicate is a
// skip (counted, no error), anything else is a hard error.
func tally(res *Result, kind string, err error, dup error) (skipped bool, fatal error) {
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, dup):
		res.Skipped[kind]++
		return true, nil
	default:
		return false, err
	}
}

// Verify checks that two archives hold the same knowledge by content — the
// round-trip guarantee. It ignores surrogate ids (public ids, uuids) that a
// restore legitimately remints, comparing stable identity + payload signatures.
func Verify(a, b *Archive) error {
	checks := []struct {
		name string
		a, b []string
	}{
		{"projects", projectSigs(a), projectSigs(b)},
		{"documents", documentSigs(a), documentSigs(b)},
		{"decisions", decisionSigs(a), decisionSigs(b)},
		{"snippets", snippetSigs(a), snippetSigs(b)},
		{"journal", journalSigs(a), journalSigs(b)},
		{"solutions", solutionSigs(a), solutionSigs(b)},
		{"memory", memorySigs(a), memorySigs(b)},
		{"env", envSigs(a), envSigs(b)},
	}
	for _, c := range checks {
		if err := sameSet(c.name, c.a, c.b); err != nil {
			return err
		}
	}
	return nil
}

func sameSet(kind string, a, b []string) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s: count %d != %d", kind, len(a), len(b))
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("%s: content mismatch: %q vs %q", kind, a[i], b[i])
		}
	}
	return nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func projectSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Projects))
	for _, p := range a.Projects {
		out = append(out, p.Slug+"\x00"+p.Name)
	}
	return out
}

func documentSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Documents))
	for _, d := range a.Documents {
		body, _ := json.Marshal(d.Body)
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s\x00%s", d.ID, d.Title, d.Status, body))
	}
	return out
}

func decisionSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Decisions))
	for _, d := range a.Decisions {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s\x00%s", deref(d.Project), d.Title, d.Decision, d.Status))
	}
	return out
}

func snippetSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Snippets))
	for _, s := range a.Snippets {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s", deref(s.Project), s.Title, s.Content))
	}
	return out
}

func journalSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Journal))
	for _, j := range a.Journal {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s", deref(j.Project), j.SessionRef, j.Summary))
	}
	return out
}

func solutionSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Solutions))
	for _, s := range a.Solutions {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s", deref(s.Project), s.ErrorDescription, s.Solution))
	}
	return out
}

func memorySigs(a *Archive) []string {
	out := make([]string, 0, len(a.Memory))
	for _, m := range a.Memory {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s", m.Scope, deref(m.Project), deref(m.Area), m.Key, m.Value))
	}
	return out
}

func envSigs(a *Archive) []string {
	out := make([]string, 0, len(a.Env))
	for _, e := range a.Env {
		out = append(out, fmt.Sprintf("%s\x00%s\x00%s", e.Project, e.Key, e.Value))
	}
	return out
}
