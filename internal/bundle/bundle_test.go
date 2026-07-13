package bundle

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// fakeStore embeds store.Store so unimplemented methods panic if called;
// the assembler only touches the six methods overridden below.
type fakeStore struct {
	store.Store
	projects  map[string]*models.Project
	memory    []*models.Memory
	documents []*models.Document
	decisions []*models.Decision
	snippets  []*models.Snippet
	journal   []*models.JournalEntry
	env       []*models.EnvEntry
}

func (f *fakeStore) GetProject(_ context.Context, slug string) (*models.Project, error) {
	if p, ok := f.projects[slug]; ok {
		return p, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) ListMemory(_ context.Context, flt store.MemoryFilter) ([]*models.Memory, error) {
	out := []*models.Memory{}
	for _, m := range f.memory {
		if flt.Scope != nil && m.Scope != *flt.Scope {
			continue
		}
		if flt.Project != nil && (m.Project == nil || *m.Project != *flt.Project) {
			continue
		}
		if flt.Area != nil && (m.Area == nil || *m.Area != *flt.Area) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (f *fakeStore) ListDocuments(_ context.Context, flt store.Filter) ([]*models.Document, error) {
	out := []*models.Document{}
	for _, d := range f.documents {
		if flt.Project != nil && (d.Project == nil || *d.Project != *flt.Project) {
			continue
		}
		if flt.Type != nil && d.Type != *flt.Type {
			continue
		}
		if flt.Status != nil && d.Status != *flt.Status {
			continue
		}
		out = append(out, d)
	}
	if flt.Limit > 0 && len(out) > flt.Limit {
		out = out[:flt.Limit]
	}
	return out, nil
}

func (f *fakeStore) ListDecisions(_ context.Context, _ store.DecisionFilter) ([]*models.Decision, error) {
	return f.decisions, nil
}
func (f *fakeStore) ListSnippets(_ context.Context, _ store.SnippetFilter) ([]*models.Snippet, error) {
	return f.snippets, nil
}
func (f *fakeStore) ListJournalEntries(_ context.Context, _ store.JournalFilter) ([]*models.JournalEntry, error) {
	return f.journal, nil
}
func (f *fakeStore) ListEnv(_ context.Context, project string) ([]*models.EnvEntry, error) {
	out := []*models.EnvEntry{}
	for _, e := range f.env {
		if e.Project == project {
			out = append(out, e)
		}
	}
	return out, nil
}

func strptr(s string) *string { return &s }

func TestAssembleMergesMemoryAndFilters(t *testing.T) {
	f := &fakeStore{
		projects: map[string]*models.Project{"mneme": {Slug: "mneme"}},
		memory: []*models.Memory{
			{Scope: models.ScopeGlobal, Key: "editor", Value: "vim"},
			{Scope: models.ScopeProject, Project: strptr("mneme"), Key: "editor", Value: "nvim"}, // overrides global
			{Scope: models.ScopeProject, Project: strptr("mneme"), Key: "db", Value: "postgres"},
		},
		decisions: []*models.Decision{
			{Title: "mneme dec", Project: strptr("mneme"), Status: "accepted"},
			{Title: "global dec", Project: nil, Status: "accepted"},
			{Title: "other dec", Project: strptr("hyperion"), Status: "accepted"}, // filtered out
		},
		snippets: []*models.Snippet{
			{Title: "mneme snip", Project: strptr("mneme")},
			{Title: "global snip", Project: nil},
			{Title: "other snip", Project: strptr("hyperion")}, // filtered out
		},
		journal: []*models.JournalEntry{
			{Summary: "mneme entry", Project: strptr("mneme")},
		},
		documents: []*models.Document{
			{ID: "p1", Title: "Impl plan", Project: strptr("mneme"), Type: models.TypePlan, Status: models.StatusInProgress, PhaseCurrent: ptr(6), PhaseTotal: ptr(9)},
		},
	}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if b.Memory["editor"] != "nvim" {
		t.Errorf("project memory should override global: %v", b.Memory)
	}
	if b.Memory["db"] != "postgres" {
		t.Errorf("missing project memory: %v", b.Memory)
	}
	if len(b.Decisions) != 2 {
		t.Errorf("want project+global decisions (2), got %d", len(b.Decisions))
	}
	if len(b.Snippets) != 2 {
		t.Errorf("want project+global snippets (2), got %d", len(b.Snippets))
	}
	if b.ActivePlan == nil || b.ActivePlan.Title != "Impl plan" {
		t.Fatalf("active plan: %+v", b.ActivePlan)
	}
	if !strings.Contains(b.Markdown, "phase 6/9") {
		t.Errorf("digest missing plan phase: %q", b.Markdown)
	}
}

func TestAssembleIncludesEnv(t *testing.T) {
	f := &fakeStore{
		projects: map[string]*models.Project{"mneme": {Slug: "mneme"}},
		env: []*models.EnvEntry{
			{Project: "mneme", Key: "API_PORT", Value: "8443", Description: strptr("https port")},
			{Project: "mneme", Key: "DB_SERVICE", Value: "postgres"},
			{Project: "other", Key: "X", Value: "y"}, // filtered out by project
		},
	}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(b.Env) != 2 {
		t.Fatalf("want 2 env entries for mneme, got %d", len(b.Env))
	}
	if !strings.Contains(b.Markdown, "## Env") ||
		!strings.Contains(b.Markdown, "API_PORT = 8443 — https port") ||
		!strings.Contains(b.Markdown, "DB_SERVICE = postgres") {
		t.Errorf("digest env section: %q", b.Markdown)
	}
}

func TestAssembleUnknownProject(t *testing.T) {
	f := &fakeStore{projects: map[string]*models.Project{}}
	_, err := New(f).Assemble(context.Background(), "ghost", nil)
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestAssembleCapsAndArea(t *testing.T) {
	decs := make([]*models.Decision, 8)
	for i := range decs {
		decs[i] = &models.Decision{Title: "d", Project: strptr("mneme"), Status: "accepted"}
	}
	f := &fakeStore{
		projects: map[string]*models.Project{"mneme": {Slug: "mneme"}},
		memory: []*models.Memory{
			{Scope: models.ScopeArea, Project: strptr("mneme"), Area: strptr("api"), Key: "port", Value: "8443"},
		},
		decisions: decs,
	}
	b, err := New(f).Assemble(context.Background(), "mneme", strptr("api"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Decisions) != 5 {
		t.Errorf("decisions should cap at 5, got %d", len(b.Decisions))
	}
	if b.Memory["port"] != "8443" {
		t.Errorf("area memory not included: %v", b.Memory)
	}
	if b.ActivePlan != nil {
		t.Errorf("no in-progress plan -> nil, got %+v", b.ActivePlan)
	}
}

func TestRenderMarkdownNoneSections(t *testing.T) {
	b := &Bundle{Project: "mneme", Memory: map[string]string{}}
	md := renderMarkdown(b)
	for _, want := range []string{
		"# Context bundle — mneme",
		"## Memory\n_none_",
		"## Env\n_none_",
		"## Active plan\n_none_",
		"## Recent journal\n_none_",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("digest missing %q:\n%s", want, md)
		}
	}
}
