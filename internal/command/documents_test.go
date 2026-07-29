package command

import (
	"context"
	"errors"
	"testing"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// countStore embeds store.Store so unimplemented methods panic; only the write
// methods the command service uses are overridden, counting their calls.
type countStore struct {
	store.Store
	creates, updates, appends int
	rev                       int
	updateErr                 error

	mentionCalls    int
	lastMentionFrom string
	lastMentions    []*models.Relation
	mentionErr      error
}

func (c *countStore) ReplaceAutoMentions(_ context.Context, fromID string, rows []*models.Relation) error {
	if c.mentionErr != nil {
		return c.mentionErr
	}
	c.mentionCalls++
	c.lastMentionFrom = fromID
	c.lastMentions = rows
	return nil
}

func (c *countStore) CreateDocument(_ context.Context, doc *models.Document) error {
	c.creates++
	c.rev = 1
	doc.Revision = 1
	return nil
}

func (c *countStore) UpdateDocument(_ context.Context, doc *models.Document, _ *int) error {
	if c.updateErr != nil {
		return c.updateErr
	}
	c.updates++
	c.rev++
	doc.Revision = c.rev
	return nil
}

func (c *countStore) AppendDocumentRevision(_ context.Context, _ *models.DocumentRevision) error {
	c.appends++
	return nil
}

type countEnq struct{ n int }

func (c *countEnq) Enqueue(embed.SourceRef) { c.n++ }

type countBC struct {
	n    int
	last live.Event
}

func (c *countBC) Broadcast(ev live.Event) { c.n++; c.last = ev }

func TestUpdateRunsSideEffectsExactlyOnce(t *testing.T) {
	st := &countStore{rev: 3}
	enq := &countEnq{}
	bc := &countBC{}
	d := NewDocuments(st, enq, bc)

	doc := &models.Document{ID: "d1", Title: "T", Status: "todo"}
	ev := live.Event{Type: "documents", ID: "d1", BlockID: "t-1", Op: "tick_task"}
	if err := d.Update(context.Background(), doc, Write{Op: "tick_task", Actor: "mcp", Event: ev}); err != nil {
		t.Fatal(err)
	}

	if st.updates != 1 {
		t.Errorf("store updates = %d, want 1", st.updates)
	}
	if st.appends != 1 {
		t.Errorf("history appends = %d, want 1", st.appends)
	}
	if enq.n != 1 {
		t.Errorf("embed enqueues = %d, want 1", enq.n)
	}
	if bc.n != 1 {
		t.Errorf("broadcasts = %d, want 1", bc.n)
	}
	if bc.last.Op != "tick_task" || bc.last.BlockID != "t-1" {
		t.Errorf("broadcast event = %+v", bc.last)
	}
}

func TestUpdateWithoutEventStillRecordsAndEmbeds(t *testing.T) {
	st := &countStore{rev: 1}
	enq := &countEnq{}
	bc := &countBC{}
	d := NewDocuments(st, enq, bc)

	// A write with an empty event still records history and enqueues re-embedding
	// but broadcasts nothing (no live event to emit).
	doc := &models.Document{ID: "d1"}
	if err := d.Update(context.Background(), doc, Write{Op: "migrate", Actor: "cli"}); err != nil {
		t.Fatal(err)
	}
	if st.appends != 1 || enq.n != 1 {
		t.Errorf("history/embed should still run once: appends=%d enq=%d", st.appends, enq.n)
	}
	if bc.n != 0 {
		t.Errorf("no event → no broadcast, got %d", bc.n)
	}
}

func TestUpdateStoreErrorSkipsSideEffects(t *testing.T) {
	st := &countStore{updateErr: errors.New("boom")}
	enq := &countEnq{}
	bc := &countBC{}
	d := NewDocuments(st, enq, bc)

	err := d.Update(context.Background(), &models.Document{ID: "d1"}, Write{Op: "x", Event: live.Event{Type: "documents", ID: "d1"}})
	if err == nil {
		t.Fatal("expected the store error to propagate")
	}
	if st.appends != 0 || enq.n != 0 || bc.n != 0 {
		t.Errorf("a failed persist must skip all side effects: appends=%d enq=%d bc=%d", st.appends, enq.n, bc.n)
	}
}

// Every document write re-syncs the doc's auto-mention edges from its body —
// the scanner half of spec-relations — and a sync failure fails the write.
func TestWriteSyncsAutoMentions(t *testing.T) {
	st := &countStore{}
	d := NewDocuments(st, nil, nil)
	doc := &models.Document{
		ID: "plan-a", PublicID: "doc_aaa111bbb", Title: "A", Type: "plan", Status: "todo",
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "see doc_abc123xyz"},
		}},
	}
	if err := d.Create(context.Background(), doc, Write{Op: "push_document", Actor: "test"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if st.mentionCalls != 1 || st.lastMentionFrom != "doc_aaa111bbb" {
		t.Fatalf("mention sync after Create: calls=%d from=%q", st.mentionCalls, st.lastMentionFrom)
	}
	if len(st.lastMentions) != 1 || st.lastMentions[0].ToRef != "doc_abc123xyz" {
		t.Fatalf("mention rows after Create: %+v", st.lastMentions)
	}

	doc.Body = map[string]any{"sections": []any{
		map[string]any{"type": "text", "id": "p", "content": "now [[other-doc]]"},
	}}
	if err := d.Update(context.Background(), doc, Write{Op: "update_section", Actor: "test"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if st.mentionCalls != 2 || len(st.lastMentions) != 1 || st.lastMentions[0].ToRef != "other-doc" {
		t.Fatalf("mention rows after Update: calls=%d rows=%+v", st.mentionCalls, st.lastMentions)
	}

	st.mentionErr = errors.New("boom")
	if err := d.Update(context.Background(), doc, Write{Op: "update_section", Actor: "test"}); err == nil {
		t.Fatal("mention sync error must propagate")
	}
}
