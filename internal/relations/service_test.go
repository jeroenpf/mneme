package relations

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

func newTestService(t *testing.T) (*Service, store.Store) {
	t.Helper()
	ctx := context.Background()
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return &Service{Store: st}, st
}

func seedDoc(t *testing.T, st store.Store, slug, title string) *models.Document {
	t.Helper()
	d := &models.Document{ID: slug, Title: title, Type: "plan", Status: "todo",
		Body: map[string]any{"sections": []any{}}}
	if err := st.CreateDocument(context.Background(), d); err != nil {
		t.Fatalf("seed %s: %v", slug, err)
	}
	return d
}

func TestServiceLinkValidation(t *testing.T) {
	svc, st := newTestService(t)
	ctx := context.Background()
	a := seedDoc(t, st, "plan-a", "Plan A")
	b := seedDoc(t, st, "plan-b", "Plan B")

	// Accepts a doc slug for from and a public id for to.
	rel, err := svc.Link(ctx, "plan-a", b.PublicID, "implements")
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	if rel.FromID != a.PublicID || rel.ToRef != b.PublicID || rel.Origin != "explicit" {
		t.Fatalf("stored relation: %+v", rel)
	}

	if _, err := svc.Link(ctx, "plan-a", b.PublicID, "mentions"); err == nil {
		t.Fatal("rel_type mentions must be rejected (scanner-owned)")
	}
	if _, err := svc.Link(ctx, "plan-a", b.PublicID, "nonsense"); err == nil {
		t.Fatal("unknown rel_type must be rejected")
	}
	if _, err := svc.Link(ctx, "plan-a", "doc_missing000", "relates-to"); err == nil {
		t.Fatal("missing target must be rejected")
	}
	if _, err := svc.Link(ctx, "plan-a", "plan-a", "relates-to"); err == nil {
		t.Fatal("self-link must be rejected")
	}

	// Decision endpoint: cross-kind links work.
	dec := &models.Decision{Title: "Big call", Decision: "yes", Status: "accepted"}
	if err := st.CreateDecision(ctx, dec); err != nil {
		t.Fatalf("seed decision: %v", err)
	}
	if _, err := svc.Link(ctx, "plan-a", dec.PublicID, "relates-to"); err != nil {
		t.Fatalf("Link to decision: %v", err)
	}
}

func TestServiceRelatedAndUnlink(t *testing.T) {
	svc, st := newTestService(t)
	ctx := context.Background()
	a := seedDoc(t, st, "plan-a", "Plan A")
	b := seedDoc(t, st, "plan-b", "Plan B")

	if _, err := svc.Link(ctx, a.PublicID, b.PublicID, "implements"); err != nil {
		t.Fatalf("Link: %v", err)
	}
	// Auto mentions: A mentions B (by public id) and the dangling [[plan-c]].
	if err := st.ReplaceAutoMentions(ctx, a.PublicID, MentionRows(&models.Document{
		ID: a.ID, PublicID: a.PublicID,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": b.PublicID + " and [[plan-c]]"},
		}},
	})); err != nil {
		t.Fatalf("seed mentions: %v", err)
	}

	// Outgoing view from A.
	got, err := svc.Related(ctx, "plan-a")
	if err != nil {
		t.Fatalf("Related(a): %v", err)
	}
	if len(got.Links) != 1 || got.Links[0].RelType != "implements" || got.Links[0].Direction != "out" ||
		got.Links[0].Title != "Plan B" || got.Links[0].DocStatus != "todo" {
		t.Fatalf("a.Links: %+v", got.Links)
	}
	if len(got.Mentions) != 2 {
		t.Fatalf("a.Mentions: %+v", got.Mentions)
	}
	var dangling *Entry
	for i := range got.Mentions {
		if got.Mentions[i].Dangling {
			dangling = &got.Mentions[i]
		}
	}
	if dangling == nil || dangling.ID != "plan-c" || dangling.Kind != "document" {
		t.Fatalf("dangling mention: %+v", got.Mentions)
	}

	// Incoming view from B: the typed link and the mention, both direction=in.
	got, err = svc.Related(ctx, b.PublicID)
	if err != nil {
		t.Fatalf("Related(b): %v", err)
	}
	if len(got.Links) != 1 || got.Links[0].Direction != "in" || got.Links[0].Title != "Plan A" {
		t.Fatalf("b.Links: %+v", got.Links)
	}
	if len(got.MentionedBy) != 1 || got.MentionedBy[0].Title != "Plan A" {
		t.Fatalf("b.MentionedBy: %+v", got.MentionedBy)
	}

	// A dangling slug becomes a live backlink once the doc exists.
	seedDoc(t, st, "plan-c", "Plan C")
	got, err = svc.Related(ctx, "plan-c")
	if err != nil {
		t.Fatalf("Related(c): %v", err)
	}
	if len(got.MentionedBy) != 1 || got.MentionedBy[0].Title != "Plan A" {
		t.Fatalf("c.MentionedBy: %+v", got.MentionedBy)
	}

	// Unlink removes the explicit edge only.
	n, err := svc.Unlink(ctx, "plan-a", "plan-b", nil)
	if err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if n != 1 {
		t.Fatalf("Unlink removed %d, want 1", n)
	}
	got, _ = svc.Related(ctx, "plan-a")
	if len(got.Links) != 0 || len(got.Mentions) != 2 {
		t.Fatalf("after unlink: links=%+v mentions=%+v", got.Links, got.Mentions)
	}
}

// A body that references the same target by slug AND public id produces two
// edge rows, but the related view collapses them: one entry per endpoint,
// rel type, and direction.
func TestServiceRelatedDedupsSameEndpoint(t *testing.T) {
	svc, st := newTestService(t)
	ctx := context.Background()
	a := seedDoc(t, st, "plan-a", "Plan A")
	b := seedDoc(t, st, "plan-b", "Plan B")

	if err := st.ReplaceAutoMentions(ctx, a.PublicID, MentionRows(&models.Document{
		ID: a.ID, PublicID: a.PublicID,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "[[plan-b]] aka " + b.PublicID},
		}},
	})); err != nil {
		t.Fatalf("seed mentions: %v", err)
	}

	got, err := svc.Related(ctx, "plan-a")
	if err != nil {
		t.Fatalf("Related(a): %v", err)
	}
	if len(got.Mentions) != 1 || got.Mentions[0].ID != b.PublicID {
		t.Fatalf("a.Mentions must dedup to one entry: %+v", got.Mentions)
	}
	got, err = svc.Related(ctx, "plan-b")
	if err != nil {
		t.Fatalf("Related(b): %v", err)
	}
	if len(got.MentionedBy) != 1 {
		t.Fatalf("b.MentionedBy must dedup to one entry: %+v", got.MentionedBy)
	}
}

func TestServiceResolveErrors(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Related(ctx, "no-such-doc"); err == nil ||
		!strings.Contains(err.Error(), "no-such-doc") {
		t.Fatalf("Related(unknown) = %v, want error naming the ref", err)
	}
	if _, err := svc.Related(ctx, ""); err == nil {
		t.Fatal("empty ref must error")
	}
}
