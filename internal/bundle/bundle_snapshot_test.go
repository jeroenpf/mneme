package bundle

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeroenpf/mneme/internal/models"
)

var updateGolden = flag.Bool("update", false, "update bundle golden snapshot files")

// fixedTime keeps rendered dates deterministic across snapshot runs.
var fixedTime = time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)

// smallFixture: a just-started project — one in-progress plan, one memory,
// one decision, one journal entry.
func smallFixture() *fakeStore {
	plan := &models.Document{
		ID: "auth", Title: "Auth service", Project: strptr("mneme"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		PhaseCurrent: ptr(2), PhaseTotal: ptr(2), Tags: []string{"auth"},
		Meta: map[string]any{"phases": []any{
			map[string]any{"title": "Schema", "status": "done"},
			map[string]any{"title": "Handlers", "status": "wip"},
		}},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-2", "title": "Handlers", "tasks": []any{
				map[string]any{"id": "t-1", "title": "add login handler", "done": false},
				map[string]any{"id": "t-2", "title": "add logout handler", "done": false},
			}},
		}},
	}
	return &fakeStore{
		projects:  map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{plan},
		memory:    []*models.Memory{{Scope: models.ScopeProject, Project: strptr("mneme"), Key: "db", Value: "postgres"}},
		decisions: []*models.Decision{{Title: "Use JWT sessions", Project: strptr("mneme"), Status: models.DecisionAccepted, Rationale: "Stateless verification fits a single-node local service.", CreatedAt: fixedTime}},
		journal:   []*models.JournalEntry{{Summary: "Kicked off the auth service plan", Project: strptr("mneme"), SessionRef: "sp-1", CreatedAt: fixedTime}},
	}
}

// typicalFixture: a project mid-flight — a richer plan, a few memories,
// decisions with rationale, snippets, journal, and a blocker.
func typicalFixture() *fakeStore {
	plan := &models.Document{
		ID: "search", Title: "Retrieval quality", Project: strptr("mneme"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		PhaseCurrent: ptr(3), PhaseTotal: ptr(5), Tags: []string{"search", "embeddings"},
		Meta: map[string]any{"phases": []any{
			map[string]any{"title": "Chunker", "status": "done"},
			map[string]any{"title": "Fusion", "status": "wip"},
			map[string]any{"title": "Navigation", "status": "todo"},
		}},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-2", "title": "Fusion", "tasks": []any{
				map[string]any{"id": "t-1", "title": "normalize cross-type scores", "done": false},
				map[string]any{"id": "t-2", "title": "add relevance fixtures", "done": false},
			}},
			map[string]any{"type": "subphase", "id": "sp-3", "title": "Navigation", "tasks": []any{
				map[string]any{"id": "t-3", "title": "deep-link every hit", "done": false},
			}},
		}},
	}
	blocked := &models.Document{ID: "provider", Title: "Local embedding provider spike", Project: strptr("mneme"), Type: models.TypeSpec, Status: models.StatusBlocked}
	return &fakeStore{
		projects:  map[string]*models.Project{"mneme": {Slug: "mneme"}},
		documents: []*models.Document{plan, blocked},
		memory: []*models.Memory{
			{Scope: models.ScopeGlobal, Key: "editor", Value: "nvim"},
			{Scope: models.ScopeProject, Project: strptr("mneme"), Key: "db", Value: "postgres"},
			{Scope: models.ScopeProject, Project: strptr("mneme"), Key: "search", Value: "hybrid fts + vectors"},
		},
		decisions: []*models.Decision{
			{Title: "Reciprocal-rank fusion", Project: strptr("mneme"), Status: models.DecisionAccepted, Rationale: "RRF blends lexical and semantic ranks without tuning per-type score scales.", CreatedAt: fixedTime},
			{Title: "Recursive AST chunker", Project: strptr("mneme"), Status: models.DecisionAccepted, Rationale: "Walking every block type indexes tasks and tables the section chunker missed.", CreatedAt: fixedTime},
		},
		snippets: []*models.Snippet{
			{Title: "cosine similarity", Project: strptr("mneme"), Language: "go", Tags: []string{"search"}, Description: "Brute-force cosine over the embedding slice."},
			{Title: "fts5 query", Project: strptr("mneme"), Language: "sql", Tags: []string{"search"}, Description: "bm25-ranked FTS5 match with a project filter."},
		},
		journal: []*models.JournalEntry{
			{Summary: "Landed the recursive chunker and stable chunk ids", Project: strptr("mneme"), SessionRef: "sp-2", Deferred: []string{"normalize fusion scores", "excerpt from matched chunks"}, CreatedAt: fixedTime},
			{Summary: "Spiked pgvector index tuning", Project: strptr("mneme"), SessionRef: "sp-1", CreatedAt: fixedTime},
		},
	}
}

// largeFixture: an established project whose full content overflows the default
// token budget, exercising deterministic truncation.
func largeFixture() *fakeStore {
	f := typicalFixture()
	longRationale := "A deliberately verbose rationale that spells out the trade-offs weighed, the alternatives " +
		"rejected, and the follow-on consequences in more than enough detail to consume a meaningful slice " +
		"of the token budget so the truncation path is exercised end to end under realistic pressure."
	longDesc := "A thoroughly documented reusable pattern with plenty of surrounding prose describing when to " +
		"reach for it, the edge cases it handles, and why it beats the obvious naive alternative."
	longSummary := "A detailed session summary covering everything built, everything consciously deferred, and every " +
		"behavioural change shipped across a long and productive day of focused implementation work on the project."
	for i := range 8 {
		f.decisions = append(f.decisions, &models.Decision{Title: "Decision " + string(rune('A'+i)), Project: strptr("mneme"), Status: models.DecisionAccepted, Rationale: longRationale, CreatedAt: fixedTime})
	}
	for i := range 12 {
		f.snippets = append(f.snippets, &models.Snippet{Title: "Pattern " + string(rune('A'+i)), Project: strptr("mneme"), Language: "go", Description: longDesc})
	}
	for i := range 5 {
		f.journal = append(f.journal, &models.JournalEntry{Summary: longSummary + " #" + string(rune('A'+i)), Project: strptr("mneme"), SessionRef: "sp-x", CreatedAt: fixedTime})
	}
	// A long deferred list (from the newest journal entry) pushes the full
	// render past the default budget so truncation is exercised.
	var deferred []string
	for i := range 10 {
		deferred = append(deferred, "backfill embeddings for legacy documents pushed before the chunker rewrite, batch "+string(rune('A'+i)))
	}
	f.journal[0].Deferred = deferred
	return f
}

func TestBundleSnapshots(t *testing.T) {
	cases := []struct {
		name          string
		store         *fakeStore
		area          *string
		maxTokens     int
		wantTruncated bool
	}{
		{"small", smallFixture(), nil, 200, false},
		{"typical", typicalFixture(), strptr("search"), 600, false},
		{"large", largeFixture(), nil, DefaultTokenBudget, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := New(tc.store).Assemble(context.Background(), "mneme", tc.area)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: %d estimated tokens (truncated=%v)", tc.name, b.EstimatedTokens, b.Truncated)

			golden := filepath.Join("testdata", tc.name+".golden.md")
			if *updateGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(b.Markdown), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run `go test -run TestBundleSnapshots -update`): %v", err)
			}
			if b.Markdown != string(want) {
				t.Errorf("digest mismatch for %s (run -update to refresh):\n--- got ---\n%s\n--- want ---\n%s", tc.name, b.Markdown, want)
			}

			// Token-size budget: each size stays under its bound, and the large
			// project is trimmed to fit the default budget.
			if b.EstimatedTokens > tc.maxTokens {
				t.Errorf("%s digest %d tokens exceeds bound %d", tc.name, b.EstimatedTokens, tc.maxTokens)
			}
			if b.Truncated != tc.wantTruncated {
				t.Errorf("%s truncated = %v, want %v", tc.name, b.Truncated, tc.wantTruncated)
			}
			if tc.wantTruncated && b.EstimatedTokens > DefaultTokenBudget {
				t.Errorf("%s still over budget after truncation: %d > %d", tc.name, b.EstimatedTokens, DefaultTokenBudget)
			}
		})
	}
}

func TestBundleSnapshotsMonotonicSize(t *testing.T) {
	small, _ := New(smallFixture()).Assemble(context.Background(), "mneme", nil)
	typical, _ := New(typicalFixture()).Assemble(context.Background(), "mneme", nil)
	if small.EstimatedTokens >= typical.EstimatedTokens {
		t.Errorf("small (%d) should be smaller than typical (%d)", small.EstimatedTokens, typical.EstimatedTokens)
	}
}
