// Package command centralizes document writes behind one validated application
// service shared by the REST API and the MCP tools (roadmap P6). It owns the
// side effects that must run exactly once per write — persistence (with the
// store's revision bump and optimistic-concurrency check), the append-only
// history/audit snapshot, embedding enqueue, and the live broadcast — so no
// write path can silently skip history or embeddings again.
package command

import (
	"context"
	"fmt"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/relations"
	"github.com/jeroenpf/mneme/internal/store"
)

// Documents is the single write path for documents.
type Documents struct {
	store store.Store
	enq   embed.Enqueuer
	bc    live.Broadcaster
}

// NewDocuments builds the service. A nil enqueuer/broadcaster is replaced with
// the no-op variant, so callers can wire it whether or not embedding/live are
// enabled.
func NewDocuments(st store.Store, enq embed.Enqueuer, bc live.Broadcaster) *Documents {
	if enq == nil {
		enq = embed.NopEnqueuer{}
	}
	if bc == nil {
		bc = live.NopBroadcaster{}
	}
	return &Documents{store: st, enq: enq, bc: bc}
}

// Write carries a mutation's audit and live metadata. Op is the operation name
// (tool or endpoint), Actor the surface ("mcp"/"rest"), Event the live event to
// broadcast (its BlockID, when set, names the changed node for the audit
// record), and Expected an optional optimistic-concurrency base revision
// (Update only).
type Write struct {
	Op       string
	Actor    string
	Event    live.Event
	Expected *int
}

// Create persists a new document (revision 1) and runs the once-per-write side
// effects. Store errors (ErrInvalidProject, ErrDuplicateID) propagate untranslated.
func (d *Documents) Create(ctx context.Context, doc *models.Document, w Write) error {
	if err := d.store.CreateDocument(ctx, doc); err != nil {
		return err
	}
	return d.afterWrite(ctx, doc, w)
}

// Update persists an existing document — bumping its revision, checking
// w.Expected when set — and runs the once-per-write side effects. Returns
// *store.RevisionConflictError on a stale expected revision; the row is untouched.
func (d *Documents) Update(ctx context.Context, doc *models.Document, w Write) error {
	if err := d.store.UpdateDocument(ctx, doc, w.Expected); err != nil {
		return err
	}
	return d.afterWrite(ctx, doc, w)
}

// afterWrite records the history snapshot, enqueues re-embedding, and broadcasts
// the live event — the three once-per-write side effects, in order.
func (d *Documents) afterWrite(ctx context.Context, doc *models.Document, w Write) error {
	rev := &models.DocumentRevision{
		DocumentID: doc.ID,
		Revision:   doc.Revision,
		Op:         w.Op,
		Actor:      w.Actor,
		TargetIDs:  eventTargets(w.Event, doc.ID),
		Title:      doc.Title,
		Status:     doc.Status,
		Meta:       doc.Meta,
		Body:       doc.Body,
	}
	if err := d.store.AppendDocumentRevision(ctx, rev); err != nil {
		return fmt.Errorf("record revision: %w", err)
	}
	// Re-sync the doc's auto-mention edges from its current body: since every
	// document mutation funnels through this write path, the relations graph
	// can never drift from the prose (spec-relations).
	if err := d.store.ReplaceAutoMentions(ctx, doc.PublicID, relations.MentionRows(doc)); err != nil {
		return fmt.Errorf("sync mentions: %w", err)
	}
	d.enq.Enqueue(embed.SourceRef{Type: "documents", ID: doc.ID})
	if w.Event.Type != "" {
		d.bc.Broadcast(w.Event)
	}
	return nil
}

// eventTargets is the affected-id list for an audit record: the edited block
// when the event names one, otherwise the whole document.
func eventTargets(ev live.Event, docID string) []string {
	if ev.BlockID != "" {
		return []string{ev.BlockID}
	}
	return []string{docID}
}
