package store_test

import (
	"context"
	"testing"

	"github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
)

// TestPublicIDColumnDefaultsToAValidID verifies migration 015: every
// top-level entity has a public_id column whose DEFAULT mints an id matching
// the internal/ids contract, so even a raw SQL insert (seed data, manual)
// gets a valid, kind-correct public id without the application supplying one.
func TestPublicIDColumnDefaultsToAValidID(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	cases := []struct {
		table  string
		insert string
		kind   ids.Kind
	}{
		{"projects", `INSERT INTO projects (name, slug) VALUES ('P','p-slug') RETURNING public_id`, ids.KindProject},
		{"documents", `INSERT INTO documents (id, title, type) VALUES ('doc-slug','T','spec') RETURNING public_id`, ids.KindDocument},
		{"decisions", `INSERT INTO decisions (title, decision) VALUES ('T','D') RETURNING public_id`, ids.KindDecision},
		{"snippets", `INSERT INTO snippets (title, content) VALUES ('T','C') RETURNING public_id`, ids.KindSnippet},
		{"journal_entries", `INSERT INTO journal_entries (summary) VALUES ('S') RETURNING public_id`, ids.KindJournal},
		{"solutions", `INSERT INTO solutions (error_description, solution) VALUES ('E','S') RETURNING public_id`, ids.KindSolution},
	}
	for _, c := range cases {
		t.Run(c.table, func(t *testing.T) {
			var pid string
			if err := s.Pool().QueryRow(ctx, c.insert).Scan(&pid); err != nil {
				t.Fatalf("insert into %s: %v", c.table, err)
			}
			if !ids.ValidFor(c.kind, pid) {
				t.Errorf("%s.public_id = %q, want a valid %s id", c.table, pid, c.kind)
			}
		})
	}
}

// TestPublicIDIsUniqueAcrossRows verifies the DEFAULT is evaluated per row
// (distinct backfill/insert values) and that the unique index is in force.
func TestPublicIDIsUniqueAcrossRows(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	var a, b string
	if err := s.Pool().QueryRow(ctx, `INSERT INTO decisions (title, decision) VALUES ('A','D') RETURNING public_id`).Scan(&a); err != nil {
		t.Fatalf("insert A: %v", err)
	}
	if err := s.Pool().QueryRow(ctx, `INSERT INTO decisions (title, decision) VALUES ('B','D') RETURNING public_id`).Scan(&b); err != nil {
		t.Fatalf("insert B: %v", err)
	}
	if a == b {
		t.Fatalf("two rows share public_id %q — default is not per-row", a)
	}

	// A duplicate value must be rejected by the unique index.
	if _, err := s.Pool().Exec(ctx, `INSERT INTO decisions (title, decision, public_id) VALUES ('C','D',$1)`, a); err == nil {
		t.Errorf("inserting a duplicate public_id %q should violate the unique index", a)
	}
}

// TestStoreThreadsPublicIDThroughCreateAndGet verifies the store reads the
// DB-minted public_id back on create (into the model) and returns it on a
// subsequent read, for both the slug-keyed (documents) and UUID-keyed tables.
func TestStoreThreadsPublicIDThroughCreateAndGet(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	doc := &models.Document{ID: "veh", Title: "T", Type: "spec", Status: models.StatusTodo}
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if !ids.ValidFor(ids.KindDocument, doc.PublicID) {
		t.Errorf("CreateDocument left public_id = %q, want a valid doc id", doc.PublicID)
	}
	if got, err := s.GetDocument(ctx, "veh"); err != nil || got.PublicID != doc.PublicID {
		t.Errorf("GetDocument public_id = %q (err %v), want %q", got.PublicID, err, doc.PublicID)
	}

	proj := &models.Project{Name: "Apollo", Slug: "apollo"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if !ids.ValidFor(ids.KindProject, proj.PublicID) {
		t.Errorf("CreateProject public_id = %q, want a valid project id", proj.PublicID)
	}

	dec := &models.Decision{Title: "T", Decision: "D", Status: models.DecisionAccepted}
	if err := s.CreateDecision(ctx, dec); err != nil {
		t.Fatalf("CreateDecision: %v", err)
	}
	if !ids.ValidFor(ids.KindDecision, dec.PublicID) {
		t.Errorf("CreateDecision public_id = %q, want a valid decision id", dec.PublicID)
	}
	if got, err := s.GetDecision(ctx, dec.ID); err != nil || got.PublicID != dec.PublicID {
		t.Errorf("GetDecision public_id = %q (err %v), want %q", got.PublicID, err, dec.PublicID)
	}

	snip := &models.Snippet{Title: "T", Content: "C"}
	if err := s.CreateSnippet(ctx, snip); err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}
	if !ids.ValidFor(ids.KindSnippet, snip.PublicID) {
		t.Errorf("CreateSnippet public_id = %q, want a valid snippet id", snip.PublicID)
	}

	jr := &models.JournalEntry{Summary: "S"}
	if err := s.CreateJournalEntry(ctx, jr); err != nil {
		t.Fatalf("CreateJournalEntry: %v", err)
	}
	if !ids.ValidFor(ids.KindJournal, jr.PublicID) {
		t.Errorf("CreateJournalEntry public_id = %q, want a valid journal id", jr.PublicID)
	}

	sol := &models.Solution{ErrorDescription: "E", Solution: "S"}
	if err := s.CreateSolution(ctx, sol); err != nil {
		t.Fatalf("CreateSolution: %v", err)
	}
	if !ids.ValidFor(ids.KindSolution, sol.PublicID) {
		t.Errorf("CreateSolution public_id = %q, want a valid solution id", sol.PublicID)
	}
}
