package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// TestJournalFTSColumn proves migration 012 added a working search_vector to
// journal_entries: an entry is findable by a word from its summary.
func TestJournalFTSColumn(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{
		Project: ptr("apollo"),
		Summary: "shipped the zigbee coordinator swap",
	}); err != nil {
		t.Fatal(err)
	}

	var n int
	err := s.Pool().QueryRow(ctx,
		`SELECT count(*) FROM journal_entries
		 WHERE search_vector @@ websearch_to_tsquery('english', $1)`,
		"zigbee").Scan(&n)
	if err != nil {
		t.Fatalf("query journal search_vector: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 journal FTS match, got %d", n)
	}
}

func seedSearchCorpus(t *testing.T, s *store.PostgresStore) {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "zigbee-plan", Title: "Zigbee migration plan", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	// The decision text carries the bare token "zigbee" so a "zigbee" query
	// matches it. Postgres tokenizes "zigbee2mqtt" as a single numword
	// lexeme that does NOT match "zigbee" (discovered during execution — see
	// the plan's note), hence the extra bare word alongside the realistic
	// product name.
	if err := s.CreateDecision(ctx, &models.Decision{
		Title: "Use zigbee2mqtt", Project: ptr("apollo"),
		Decision: "adopt zigbee2mqtt for the zigbee mesh", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSnippet(ctx, &models.Snippet{
		Title: "zigbee pairing helper", Project: ptr("apollo"),
		Language: "go", Content: "// zigbee pair", Tags: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSolution(ctx, &models.Solution{
		Project: ptr("apollo"), ErrorDescription: "zigbee coordinator offline",
		Solution: "reflash the stick", Tags: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{
		Project: ptr("apollo"), Summary: "zigbee coordinator swap done",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMemory(ctx, &models.Memory{
		Scope: models.ScopeProject, Project: ptr("apollo"),
		Key: "zigbee-note", Value: "zigbee mesh coordinator facts",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchSpansAllTypes(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s)

	hits, err := s.Search(context.Background(), "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	got := map[string]bool{}
	for _, h := range hits {
		got[h.Type] = true
		if h.Title == "" || h.Score <= 0 {
			t.Errorf("bad hit: %+v", h)
		}
	}
	for _, ty := range store.SearchTypes {
		if !got[ty] {
			t.Errorf("expected a %s hit, got types %v", ty, got)
		}
	}
}

func TestSearchTypeAndProjectFilter(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo", "hyperion")
	seedSearchCorpus(t, s)
	// a decision under a different project that must be filtered out by project
	if err := s.CreateDecision(context.Background(), &models.Decision{
		Title: "zigbee elsewhere", Project: ptr("hyperion"),
		Decision: "zigbee", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}

	// types filter: only decisions
	hits, err := s.Search(context.Background(), "zigbee",
		store.SearchFilter{Types: []string{"decisions"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.Type != "decisions" {
			t.Errorf("types filter leaked: %+v", h)
		}
	}
	// project filter: only apollo decisions
	hits, err = s.Search(context.Background(), "zigbee",
		store.SearchFilter{Types: []string{"decisions"}, Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		// apollo or global (nil) is in scope; another project is a leak.
		if h.Project != nil && *h.Project != "apollo" {
			t.Errorf("project filter leaked a foreign-project hit: %+v", h)
		}
	}
}

func TestSearchUnknownTypeErrors(t *testing.T) {
	s := newStore(t)
	_, err := s.Search(context.Background(), "x", store.SearchFilter{Types: []string{"bogus"}})
	if !errors.Is(err, store.ErrInvalidSearchType) {
		t.Fatalf("expected ErrInvalidSearchType, got %v", err)
	}
}

func TestSearchLimit(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s)
	hits, err := s.Search(context.Background(), "zigbee", store.SearchFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("limit=2 should cap at 2, got %d", len(hits))
	}
}

func TestSearchHybridFusesVectorAndFTS(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// A doc that does NOT contain the FTS term, reachable only by vector.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "sem-only", Title: "Wireless mesh notes", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	// Give it an embedding near the query vector.
	qv := fakeVec(0.9)
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{{
		SourceType: "documents", SourceID: "sem-only", ChunkID: "full",
		ChunkText: "wireless mesh coordinator", Embedding: qv,
		Project: ptr("apollo"), SourceTitle: "Wireless mesh notes", Model: "fake",
	}}); err != nil {
		t.Fatal(err)
	}

	// FTS-only (nil vector): 'zigbee' does not match "Wireless mesh notes".
	fts, err := s.Search(ctx, "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range fts {
		if h.ID == "sem-only" {
			t.Fatal("FTS-only should not surface the semantic-only doc")
		}
	}

	// Hybrid: with the query vector, the semantic-only doc appears.
	hyb, err := s.Search(ctx, "zigbee", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, h := range hyb {
		if h.ID == "sem-only" {
			found = true
			if h.Title == "" || h.Excerpt == "" || h.Score <= 0 {
				t.Errorf("vector-only hit missing projected fields: %+v", h)
			}
		}
	}
	if !found {
		t.Fatalf("hybrid search should surface the semantic-only doc, got %+v", hyb)
	}
}

func TestSearchNilVectorMatchesFTSPath(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s) // from 2.8a
	a, err := s.Search(context.Background(), "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) == 0 {
		t.Fatal("expected FTS hits for zigbee")
	}
}

// orthoVec is a unit vector along one axis. Unlike fakeVec (constant value),
// two orthoVecs on different axes are cosine-orthogonal (distance 1.0), so
// they exercise the relevance floor; fakeVec vectors are all cosine-identical
// (distance 0) regardless of seed.
func orthoVec(dim int) []float32 {
	v := make([]float32, 1024)
	v[dim] = 1
	return v
}

func TestSearchSurfacesSimilarity(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// FTS-only hit: a doc with the query word but no embedding.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "fts-doc", Title: "zigbee coordinator notes", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	// Vector-only hit: reachable by embedding (identical to the query vector),
	// title carries no query word. The source row must exist — vector search
	// only surfaces live sources.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "vec-doc", Title: "Vector doc", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	qv := fakeVec(0.5)
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{{
		SourceType: "documents", SourceID: "vec-doc", ChunkID: "full",
		ChunkText: "semantic match", Embedding: qv, Project: ptr("apollo"),
		SourceTitle: "Vector doc", Model: "fake",
	}}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, "zigbee", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	var vec, fts *models.SearchHit
	for _, h := range hits {
		switch h.ID {
		case "vec-doc":
			vec = h
		case "fts-doc":
			fts = h
		}
	}
	if vec == nil || vec.Similarity == nil {
		t.Fatalf("vector hit should carry a similarity, got %+v", vec)
	}
	if *vec.Similarity < 0.99 {
		t.Errorf("identical vectors should have similarity ~1.0, got %v", *vec.Similarity)
	}
	if fts == nil {
		t.Fatalf("expected the FTS-only doc among hits")
	}
	if fts.Similarity != nil {
		t.Errorf("FTS-only hit should have nil similarity, got %v", *fts.Similarity)
	}
}

// A deleted source must vanish from both vector search and coverage even
// while its orphaned vector still sits in the embeddings table (i.e. before
// the reconcile sweep runs). The vector path must join to live sources, just
// as the FTS path reads live tables.
func TestSearchExcludesDeletedSourceVectors(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// A live doc reachable only by vector (its title carries no FTS term).
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "doomed", Title: "Wireless mesh notes", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	qv := fakeVec(0.7)
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{{
		SourceType: "documents", SourceID: "doomed", ChunkID: "full",
		ChunkText: "wireless mesh coordinator", Embedding: qv,
		Project: ptr("apollo"), SourceTitle: "Wireless mesh notes", Model: "fake",
	}}); err != nil {
		t.Fatal(err)
	}

	has := func(hits []*models.SearchHit, id string) bool {
		for _, h := range hits {
			if h.ID == id {
				return true
			}
		}
		return false
	}

	// Sanity: while live, the vector-only hit surfaces (FTS term matches nothing).
	before, err := s.Search(ctx, "zzznomatchword", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	if !has(before, "doomed") {
		t.Fatalf("precondition: live source should surface via vector, got %+v", before)
	}

	// Delete the source row, leaving the vector orphaned (no sweep has run).
	if _, err := s.Pool().Exec(ctx, `DELETE FROM documents WHERE id=$1`, "doomed"); err != nil {
		t.Fatal(err)
	}

	after, err := s.Search(ctx, "zzznomatchword", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	if has(after, "doomed") {
		t.Fatalf("deleted source surfaced in vector search: %+v", after)
	}
	cov, err := s.EmbeddingStatus(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cov {
		if c.Type == "documents" && c.Embedded != 0 {
			t.Fatalf("deleted source still counted in coverage: %+v", c)
		}
	}
}

// A document FTS hit's excerpt comes from the best-matching chunk text, not
// the raw JSON body: it is clean prose with the matched term highlighted.
func TestSearchDocumentExcerptFromChunkNotJSON(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// The matching prose lives inside a section block, so body::text is JSON.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "doc-x", Title: "Ops runbook", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "s1", "title": "Recovery",
				"content": "restore the zigbee coordinator from backup"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	// The chunk the excerpt should be drawn from.
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{{
		SourceType: "documents", SourceID: "doc-x", ChunkID: "s1",
		ChunkText: "Ops runbook | apollo | Recovery | restore the zigbee coordinator from backup",
		Embedding: fakeVec(0.1), Project: ptr("apollo"), SourceTitle: "Ops runbook", Model: "fake",
	}}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var hit *models.SearchHit
	for _, h := range hits {
		if h.ID == "doc-x" {
			hit = h
		}
	}
	if hit == nil {
		t.Fatalf("expected doc-x among hits, got %+v", hits)
	}
	// No raw body JSON leaks into the excerpt.
	for _, junk := range []string{`"type"`, `"sections"`, "{", "}"} {
		if strings.Contains(hit.Excerpt, junk) {
			t.Errorf("excerpt still contains raw body JSON %q: %q", junk, hit.Excerpt)
		}
	}
	// The matched term is highlighted from the clean chunk text.
	if !strings.Contains(hit.Excerpt, "<<zigbee>>") {
		t.Errorf("excerpt should highlight the matched term from chunk text, got %q", hit.Excerpt)
	}
}

// A document that matches FTS but has no embedded chunk falls back to a clean
// title excerpt — never raw JSON, never an error.
func TestSearchDocumentExcerptFallsBackToTitle(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "no-embed", Title: "zigbee unembedded notes", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo,
		Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var hit *models.SearchHit
	for _, h := range hits {
		if h.ID == "no-embed" {
			hit = h
		}
	}
	if hit == nil {
		t.Fatalf("expected no-embed among hits, got %+v", hits)
	}
	if strings.Contains(hit.Excerpt, "{") || hit.Excerpt == "" {
		t.Errorf("fallback excerpt should be clean title text, got %q", hit.Excerpt)
	}
}

// Memory is part of unified retrieval (FTS), but env is intentionally kept out
// — env is looked up exactly by key, never fuzzily searched.
func TestSearchIncludesMemoryNotEnv(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.SetMemory(ctx, &models.Memory{
		Scope: models.ScopeProject, Project: ptr("apollo"),
		Key: "deploy-target", Value: "runs on the zigbee gateway host",
	}); err != nil {
		t.Fatal(err)
	}
	// An env entry carrying the same term must never surface in search.
	if err := s.SetEnv(ctx, &models.EnvEntry{
		Project: "apollo", Key: "ZIGBEE_HOST", Value: "zigbee.local",
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var memHit *models.SearchHit
	for _, h := range hits {
		if h.Type == "env" {
			t.Errorf("env leaked into unified search: %+v", h)
		}
		if h.Type == "memory" {
			memHit = h
		}
	}
	if memHit == nil {
		t.Fatalf("expected a memory hit for zigbee, got %+v", hits)
	}
	if !strings.Contains(memHit.Title, "deploy-target") {
		t.Errorf("memory hit title = %q, want the key", memHit.Title)
	}
	if !strings.Contains(memHit.Excerpt, "zigbee") {
		t.Errorf("memory hit excerpt = %q, want the value with the term", memHit.Excerpt)
	}
}

// "memory" is a valid search type; "env" is not.
func TestSearchMemoryTypeValidEnvTypeInvalid(t *testing.T) {
	s := newStore(t)
	if _, err := s.Search(context.Background(), "x", store.SearchFilter{Types: []string{"memory"}}); err != nil {
		t.Errorf("memory should be a valid search type: %v", err)
	}
	if _, err := s.Search(context.Background(), "x", store.SearchFilter{Types: []string{"env"}}); !errors.Is(err, store.ErrInvalidSearchType) {
		t.Errorf("env must not be a searchable type, got %v", err)
	}
}

func TestSearchVectorFloorDropsWeakMatches(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// Two semantic-only sources — the query string matches neither via FTS,
	// so only the vector path can surface them. Both need live source rows;
	// vector search excludes orphaned vectors.
	for _, id := range []string{"near", "far"} {
		if err := s.CreateDocument(ctx, &models.Document{
			ID: id, Title: id, Project: ptr("apollo"),
			Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "near", ChunkID: "full", ChunkText: "near",
			Embedding: orthoVec(0), Project: ptr("apollo"), SourceTitle: "Near", Model: "fake"},
		{SourceType: "documents", SourceID: "far", ChunkID: "full", ChunkText: "far",
			Embedding: orthoVec(5), Project: ptr("apollo"), SourceTitle: "Far", Model: "fake"},
	}); err != nil {
		t.Fatal(err)
	}
	qv := orthoVec(0) // cosine distance 0 to "near", 1.0 to "far"

	has := func(hits []*models.SearchHit, id string) bool {
		for _, h := range hits {
			if h.ID == id {
				return true
			}
		}
		return false
	}

	// Floor disabled (default): both semantic-only sources surface.
	off, err := s.Search(ctx, "zzznomatchword", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	if !has(off, "near") || !has(off, "far") {
		t.Fatalf("floor off should surface both, got %+v", off)
	}

	// Floor at 0.45: the far (orthogonal, distance 1.0) source is dropped,
	// the near (distance 0) source stays.
	s.SetSearchMaxDist(0.45)
	on, err := s.Search(ctx, "zzznomatchword", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	if !has(on, "near") {
		t.Fatalf("floor should keep the near match, got %+v", on)
	}
	if has(on, "far") {
		t.Fatalf("floor should drop the far (orthogonal) match, got %+v", on)
	}
}

func hitIDs(hits []*models.SearchHit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.ID
	}
	return out
}

// A project-scoped search returns that project's content plus global
// (unscoped) content, but never another project's — applied consistently.
func TestSearchProjectScopeIncludesGlobal(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo", "hyperion")
	mkDecision := func(proj *string, title string) {
		if err := s.CreateDecision(ctx, &models.Decision{
			Title: title, Project: proj, Decision: "adopt the zigbee mesh", Status: models.DecisionAccepted,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mkDecision(nil, "global zigbee policy")
	mkDecision(ptr("apollo"), "apollo zigbee choice")
	mkDecision(ptr("hyperion"), "hyperion zigbee choice")

	hits, err := s.Search(ctx, "zigbee",
		store.SearchFilter{Types: []string{"decisions"}, Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	titles := map[string]bool{}
	for _, h := range hits {
		titles[h.Title] = true
		if h.Project != nil && *h.Project != "apollo" {
			t.Errorf("foreign-project hit leaked into apollo scope: %+v", h)
		}
	}
	if !titles["apollo zigbee choice"] {
		t.Errorf("expected the apollo decision, got %v", hitIDs(hits))
	}
	if !titles["global zigbee policy"] {
		t.Errorf("expected the global decision to surface under project scope, got %v", hitIDs(hits))
	}
	if titles["hyperion zigbee choice"] {
		t.Errorf("a foreign-project decision must not surface")
	}
}

// --- Relevance fixtures: lexical, semantic, hybrid --------------------------

func relDoc(t *testing.T, s *store.PostgresStore, id, title, content string) {
	t.Helper()
	if err := s.CreateDocument(context.Background(), &models.Document{
		ID: id, Title: title, Project: ptr("apollo"), Type: models.TypePlan,
		Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "s", "title": title, "content": content}}},
	}); err != nil {
		t.Fatal(err)
	}
}

// Lexical: a keyword query ranks the document containing that keyword above an
// unrelated one.
func TestRelevanceLexical(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	relDoc(t, s, "relevant", "Zigbee pairing", "how to pair a zigbee device")
	relDoc(t, s, "unrelated", "Cooking pasta", "boil water and add salt")

	hits, err := s.Search(context.Background(), "zigbee pairing",
		store.SearchFilter{Types: []string{"documents"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].ID != "relevant" {
		t.Fatalf("lexical: expected 'relevant' first, got %v", hitIDs(hits))
	}
}

// Semantic: with a query vector, the nearest-embedding document wins even
// though neither candidate contains the query term.
func TestRelevanceSemantic(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	for _, id := range []string{"near", "far"} {
		relDoc(t, s, id, id+" notes", "no keyword here")
	}
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "near", ChunkID: "full", ChunkText: "near",
			Embedding: orthoVec(0), Project: ptr("apollo"), SourceTitle: "near notes", Model: "fake"},
		{SourceType: "documents", SourceID: "far", ChunkID: "full", ChunkText: "far",
			Embedding: orthoVec(5), Project: ptr("apollo"), SourceTitle: "far notes", Model: "fake"},
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, "zzznomatch", store.SearchFilter{Vector: orthoVec(0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].ID != "near" {
		t.Fatalf("semantic: expected 'near' first by similarity, got %v", hitIDs(hits))
	}
}

// Hybrid: a document reachable through both channels outranks documents
// reachable through only one.
func TestRelevanceHybrid(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	relDoc(t, s, "both", "both", "zigbee coordinator")
	relDoc(t, s, "lexical", "lexical", "zigbee only here")
	relDoc(t, s, "semantic", "semantic", "unrelated content")
	q := orthoVec(0)
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "both", ChunkID: "full", ChunkText: "zigbee coordinator",
			Embedding: q, Project: ptr("apollo"), SourceTitle: "both", Model: "fake"},
		{SourceType: "documents", SourceID: "semantic", ChunkID: "full", ChunkText: "unrelated content",
			Embedding: q, Project: ptr("apollo"), SourceTitle: "semantic", Model: "fake"},
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, "zigbee", store.SearchFilter{Vector: q})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].ID != "both" {
		t.Fatalf("hybrid: doc matching both channels should rank first, got %v", hitIDs(hits))
	}
}
