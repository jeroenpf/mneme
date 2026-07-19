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

func TestAssembleExtractsNextTasksPhaseBlockersDeferred(t *testing.T) {
	plan := &models.Document{
		ID: "impl", Title: "Impl plan", Project: strptr("mneme"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		PhaseCurrent: ptr(2), PhaseTotal: ptr(3),
		Meta: map[string]any{"phases": []any{
			map[string]any{"title": "Foundation", "status": "done"},
			map[string]any{"title": "API Layer", "status": "wip"},
			map[string]any{"title": "Frontend", "status": "todo"},
		}},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-1", "title": "Foundation", "tasks": []any{
				map[string]any{"id": "t-1", "title": "done thing", "done": true},
			}},
			map[string]any{"type": "subphase", "id": "sp-2", "title": "API Layer", "tasks": []any{
				map[string]any{"id": "t-2", "title": "wire routes", "done": false},
				map[string]any{"id": "t-3", "title": "add handlers", "done": false},
			}},
		}},
	}
	blocked := &models.Document{
		ID: "spike", Title: "TLS spike", Project: strptr("mneme"),
		Type: models.TypeSpec, Status: models.StatusBlocked,
	}
	f := &fakeStore{
		projects:  map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{plan, blocked},
		journal: []*models.JournalEntry{
			{Summary: "did stuff", Project: strptr("mneme"), Deferred: []string{"backfill embeddings", "retry flaky test"}},
		},
	}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatal(err)
	}

	if b.ActivePlan == nil || b.ActivePlan.ActivePhase != "API Layer" {
		t.Errorf("active phase = %+v", b.ActivePlan)
	}
	if len(b.NextTasks) != 2 {
		t.Fatalf("next tasks = %d, want 2 incomplete: %+v", len(b.NextTasks), b.NextTasks)
	}
	if b.NextTasks[0].ID != "t-2" || b.NextTasks[0].Title != "wire routes" || b.NextTasks[0].Phase != "API Layer" {
		t.Errorf("first next task = %+v", b.NextTasks[0])
	}
	if len(b.Blockers) != 1 || b.Blockers[0].ID != "spike" {
		t.Errorf("blockers = %+v", b.Blockers)
	}
	if len(b.Deferred) != 2 || b.Deferred[0] != "backfill embeddings" {
		t.Errorf("deferred = %+v", b.Deferred)
	}
	for _, want := range []string{"t-2", "wire routes", "API Layer", "TLS spike", "backfill embeddings"} {
		if !strings.Contains(b.Markdown, want) {
			t.Errorf("digest missing %q:\n%s", want, b.Markdown)
		}
	}
}

func TestRenderIncludesDecisionRationaleAndSnippetExcerpt(t *testing.T) {
	longRationale := "Postgres is already the store of record and pgvector keeps embeddings colocated, " +
		"so a second datastore would add operational surface with no retrieval benefit for a single-user local tool " +
		"that never approaches Postgres scaling limits."
	b := &Bundle{
		Project: "mneme",
		Memory:  map[string]string{},
		Decisions: []*models.Decision{
			{Title: "Use pgvector", Status: models.DecisionAccepted, Rationale: longRationale},
			{Title: "No rationale dec", Status: models.DecisionAccepted},
		},
		Snippets: []*models.Snippet{
			{Title: "pgx pool", Language: "go", Description: "Bounded connection pool wired from config."},
			{Title: "raw content only", Language: "sql", Content: "SELECT 1\nFROM docs\nWHERE id = $1"},
		},
	}
	md := renderMarkdown(b)

	// Decision rationale appears as an excerpt, truncated with an ellipsis.
	if !strings.Contains(md, "Postgres is already the store of record") {
		t.Errorf("decision rationale excerpt missing:\n%s", md)
	}
	if !strings.Contains(md, "…") {
		t.Errorf("long rationale should be truncated with an ellipsis:\n%s", md)
	}
	if strings.Contains(md, "scaling limits.") {
		t.Errorf("long rationale should be truncated, not shown in full:\n%s", md)
	}
	// A decision without rationale still renders its title line, no dangling excerpt.
	if !strings.Contains(md, "No rationale dec") {
		t.Errorf("decision without rationale should still render:\n%s", md)
	}
	// Snippet excerpt prefers the description.
	if !strings.Contains(md, "Bounded connection pool wired from config.") {
		t.Errorf("snippet description excerpt missing:\n%s", md)
	}
	// Snippet with no description falls back to a content excerpt on one line.
	if !strings.Contains(md, "SELECT 1 FROM docs") {
		t.Errorf("snippet content excerpt (newlines collapsed) missing:\n%s", md)
	}
}

// budgetFixture builds a store whose full render is comfortably over a small
// budget: an in-progress plan (core), plus many snippets/decisions/journal
// (expendable) and one blocker.
func budgetFixture() *fakeStore {
	plan := &models.Document{
		ID: "impl", Title: "Impl plan", Project: strptr("mneme"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		PhaseCurrent: ptr(2), PhaseTotal: ptr(3),
		Meta: map[string]any{"phases": []any{
			map[string]any{"title": "API Layer", "status": "wip"},
		}},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-2", "title": "API Layer", "tasks": []any{
				map[string]any{"id": "t-2", "title": "wire routes", "done": false},
				map[string]any{"id": "t-3", "title": "add handlers", "done": false},
			}},
		}},
	}
	blocked := &models.Document{ID: "spike", Title: "TLS spike", Project: strptr("mneme"), Type: models.TypeSpec, Status: models.StatusBlocked}
	decs := make([]*models.Decision, 5)
	for i := range decs {
		decs[i] = &models.Decision{Title: "Decision number " + string(rune('A'+i)), Project: strptr("mneme"), Status: models.DecisionAccepted, Rationale: "A fairly wordy rationale that eats budget and should be trimmed when space is tight."}
	}
	snips := make([]*models.Snippet, 8)
	for i := range snips {
		snips[i] = &models.Snippet{Title: "Snippet " + string(rune('A'+i)), Project: strptr("mneme"), Language: "go", Description: "A reusable pattern worth keeping around for later reference and reuse."}
	}
	jrnl := make([]*models.JournalEntry, 3)
	for i := range jrnl {
		jrnl[i] = &models.JournalEntry{Summary: "Session summary number " + string(rune('A'+i)), Project: strptr("mneme"), SessionRef: "sp-" + string(rune('A'+i))}
	}
	return &fakeStore{
		projects:  map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{plan, blocked},
		decisions: decs,
		snippets:  snips,
		journal:   jrnl,
	}
}

func TestAssembleDefaultBudgetNoTruncation(t *testing.T) {
	b, err := New(budgetFixture()).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatal(err)
	}
	if b.TokenBudget != DefaultTokenBudget {
		t.Errorf("token budget = %d, want default %d", b.TokenBudget, DefaultTokenBudget)
	}
	if b.EstimatedTokens == 0 {
		t.Errorf("estimated tokens should be reported, got 0")
	}
	if b.Truncated {
		t.Errorf("small fixture under the default budget should not be truncated")
	}
	if len(b.Snippets) != 8 {
		t.Errorf("no truncation expected, snippets = %d", len(b.Snippets))
	}
}

func TestAssembleTightBudgetTruncatesByPriority(t *testing.T) {
	b, err := New(budgetFixture()).AssembleWithOptions(context.Background(), "mneme", nil, Options{TokenBudget: 130})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Truncated {
		t.Fatalf("tight budget should truncate; estimated=%d budget=%d", b.EstimatedTokens, b.TokenBudget)
	}
	// Expendable sections are trimmed in priority order: snippets first, then
	// journal and decisions toward their floors. Core survives.
	if len(b.Snippets) != 0 {
		t.Errorf("snippets are sacrificed first, got %d remaining", len(b.Snippets))
	}
	if len(b.Journal) < 1 {
		t.Errorf("journal floor is 1, got %d", len(b.Journal))
	}
	if len(b.Decisions) < 1 {
		t.Errorf("decision floor is 1, got %d", len(b.Decisions))
	}
	if b.ActivePlan == nil || len(b.NextTasks) != 2 {
		t.Errorf("core (plan + next tasks) must survive truncation: plan=%+v tasks=%d", b.ActivePlan, len(b.NextTasks))
	}
	if b.EstimatedTokens > b.TokenBudget {
		t.Errorf("estimated %d should fit budget %d after truncation:\n%s", b.EstimatedTokens, b.TokenBudget, b.Markdown)
	}
}

func TestAssembleSelectsRelevantSnippetsByAreaAndRecentWork(t *testing.T) {
	// 10 filler snippets (fills maxSnippets) followed by 2 that match the
	// active plan / area terms. First-N-by-order would drop the relevant ones;
	// relevance selection must surface them.
	var snips []*models.Snippet
	for i := range maxSnippets {
		snips = append(snips, &models.Snippet{Title: "filler " + string(rune('a'+i)), Project: strptr("mneme"), Description: "unrelated boilerplate helper"})
	}
	snips = append(snips,
		&models.Snippet{Title: "vector cosine", Project: strptr("mneme"), Tags: []string{"search"}, Description: "brute-force cosine similarity for search ranking"},
		&models.Snippet{Title: "fts5 ranking", Project: strptr("mneme"), Tags: []string{"search"}, Description: "bm25 lexical search scoring"},
	)

	plan := &models.Document{
		ID: "impl", Title: "Retrieval quality", Project: strptr("mneme"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Tags: []string{"search", "embeddings"},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-1", "title": "Search", "tasks": []any{
				map[string]any{"id": "t-1", "title": "tune fusion", "done": false},
			}},
		}},
	}
	f := &fakeStore{
		projects:  map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{plan},
		snippets:  snips,
	}
	b, err := New(f).Assemble(context.Background(), "mneme", strptr("search"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Snippets) != maxSnippets {
		t.Fatalf("snippets should cap at %d, got %d", maxSnippets, len(b.Snippets))
	}
	if b.Snippets[0].Title != "vector cosine" || b.Snippets[1].Title != "fts5 ranking" {
		t.Errorf("relevant snippets should rank first, got %q, %q", b.Snippets[0].Title, b.Snippets[1].Title)
	}
	// Both relevant snippets survive the cap despite appearing last in store order.
	var titles []string
	for _, s := range b.Snippets {
		titles = append(titles, s.Title)
	}
	joined := strings.Join(titles, ",")
	if !strings.Contains(joined, "vector cosine") || !strings.Contains(joined, "fts5 ranking") {
		t.Errorf("relevant snippets dropped by the cap: %s", joined)
	}
}

func TestSelectRelevantSnippetsKeepsOrderWithoutSignal(t *testing.T) {
	// With no relevance terms (all scores 0), selection preserves store order.
	snips := []*models.Snippet{
		{Title: "first"}, {Title: "second"}, {Title: "third"},
	}
	got := selectRelevantSnippets(snips, map[string]bool{}, 10)
	if len(got) != 3 || got[0].Title != "first" || got[2].Title != "third" {
		t.Errorf("stable order not preserved without signal: %+v", got)
	}
}

func TestAssembleCompletedWorkFallback(t *testing.T) {
	f := &fakeStore{
		projects: map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{
			{ID: "p1", Title: "Done plan", Project: strptr("mneme"), Type: models.TypePlan, Status: models.StatusComplete},
		},
	}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatal(err)
	}
	if b.ActivePlan != nil {
		t.Fatalf("no in-progress plan expected, got %+v", b.ActivePlan)
	}
	if !strings.Contains(b.Markdown, "all plans complete") {
		t.Errorf("completed-work fallback missing:\n%s", b.Markdown)
	}
}

func TestAssembleTodoWaitingFallback(t *testing.T) {
	f := &fakeStore{
		projects: map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{
			{ID: "p1", Title: "Next plan", Project: strptr("mneme"), Type: models.TypePlan, Status: models.StatusTodo},
		},
	}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Markdown, "todo plan") {
		t.Errorf("todo-waiting fallback missing:\n%s", b.Markdown)
	}
}

func TestAssembleEmptyProjectFallbacks(t *testing.T) {
	f := &fakeStore{projects: map[string]*models.Project{"mneme": {Slug: "mneme"}}}
	b, err := New(f).Assemble(context.Background(), "mneme", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Markdown, "no recorded work yet") {
		t.Errorf("empty-project note missing:\n%s", b.Markdown)
	}
	if !strings.Contains(b.Markdown, "no plans yet") {
		t.Errorf("no-plans fallback missing:\n%s", b.Markdown)
	}
}

func TestRenderMarkdownNoneSections(t *testing.T) {
	b := &Bundle{Project: "mneme", Memory: map[string]string{}}
	md := renderMarkdown(b)
	for _, want := range []string{
		"# Context bundle — mneme",
		"## Memory\n_none_",
		"## Env\n_none_",
		"## Active plan\n_no plans yet",
		"## Recent journal\n_none_",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("digest missing %q:\n%s", want, md)
		}
	}
}
