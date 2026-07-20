package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// --- get_document_history ---------------------------------------------

type historyInput struct {
	DocID string `json:"doc_id" jsonschema:"document id (slug or doc_ public id)"`
	Limit int    `json:"limit,omitempty" jsonschema:"max revisions (newest first); default all"`
}

// revisionSummary is one history entry without the body snapshot — a lean audit
// line. Use diff_document_revisions or restore_document_revision to work with a
// specific revision's content.
type revisionSummary struct {
	Revision  int       `json:"revision"`
	Op        string    `json:"op"`
	Actor     string    `json:"actor"`
	TargetIDs []string  `json:"target_ids"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type historyOutput struct {
	Revisions []revisionSummary `json:"revisions"`
}

func (t *tools) getDocumentHistory(ctx context.Context, _ *sdk.CallToolRequest, in historyInput) (*sdk.CallToolResult, *historyOutput, error) {
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	revs, err := t.store.ListDocumentRevisions(ctx, doc.ID, in.Limit)
	if err != nil {
		return nil, nil, err
	}
	out := &historyOutput{Revisions: make([]revisionSummary, 0, len(revs))}
	for _, r := range revs {
		out.Revisions = append(out.Revisions, revisionSummary{
			Revision: r.Revision, Op: r.Op, Actor: r.Actor, TargetIDs: r.TargetIDs,
			Title: r.Title, Status: r.Status, CreatedAt: r.CreatedAt,
		})
	}
	return nil, out, nil
}

// --- diff_document_revisions ------------------------------------------

type diffInput struct {
	DocID string `json:"doc_id" jsonschema:"document id (slug or doc_ public id)"`
	From  int    `json:"from_revision" jsonschema:"base revision to compare from"`
	To    int    `json:"to_revision,omitempty" jsonschema:"revision to compare to; omit for the current document"`
}

// diffOutput reports what changed between two revisions without re-emitting the
// documents: which node ids were added, removed, or modified, plus whether the
// document's title or status changed.
type diffOutput struct {
	FromRevision  int      `json:"from_revision"`
	ToRevision    int      `json:"to_revision"`
	TitleChanged  bool     `json:"title_changed"`
	StatusChanged bool     `json:"status_changed"`
	AddedIDs      []string `json:"added_ids"`
	RemovedIDs    []string `json:"removed_ids"`
	ModifiedIDs   []string `json:"modified_ids"`
}

func (t *tools) diffDocumentRevisions(ctx context.Context, _ *sdk.CallToolRequest, in diffInput) (*sdk.CallToolResult, *diffOutput, error) {
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	if in.From <= 0 {
		return nil, nil, errors.New("from_revision is required")
	}
	from, err := t.store.GetDocumentRevision(ctx, doc.ID, in.From)
	if err != nil {
		return nil, nil, revisionErr(err, in.From)
	}

	// "to" is a recorded revision, or the current live document when omitted.
	toTitle, toStatus, toBody, toRev := doc.Title, doc.Status, doc.Body, doc.Revision
	if in.To > 0 {
		to, err := t.store.GetDocumentRevision(ctx, doc.ID, in.To)
		if err != nil {
			return nil, nil, revisionErr(err, in.To)
		}
		toTitle, toStatus, toBody, toRev = to.Title, to.Status, to.Body, to.Revision
	}

	fromNodes := flattenNodes(from.Body)
	toNodes := flattenNodes(toBody)
	out := &diffOutput{
		FromRevision:  from.Revision,
		ToRevision:    toRev,
		TitleChanged:  from.Title != toTitle,
		StatusChanged: from.Status != toStatus,
		AddedIDs:      []string{},
		RemovedIDs:    []string{},
		ModifiedIDs:   []string{},
	}
	for id, toJSON := range toNodes {
		fromJSON, ok := fromNodes[id]
		switch {
		case !ok:
			out.AddedIDs = append(out.AddedIDs, id)
		case fromJSON != toJSON:
			out.ModifiedIDs = append(out.ModifiedIDs, id)
		}
	}
	for id := range fromNodes {
		if _, ok := toNodes[id]; !ok {
			out.RemovedIDs = append(out.RemovedIDs, id)
		}
	}
	sort.Strings(out.AddedIDs)
	sort.Strings(out.RemovedIDs)
	sort.Strings(out.ModifiedIDs)
	return nil, out, nil
}

// --- restore_document_revision ----------------------------------------

type restoreInput struct {
	DocID     string `json:"doc_id" jsonschema:"document id (slug or doc_ public id)"`
	Revision  int    `json:"revision" jsonschema:"the revision whose content to restore"`
	ReturnDoc bool   `json:"return_doc,omitempty" jsonschema:"when true, also return the full restored document"`
}

// restoreResult confirms a restore: the document summary plus the new revision
// the restore produced (history is forward-only — restoring writes a new
// revision rather than rewriting the past).
type restoreResult struct {
	docSummary
	RestoredFrom int              `json:"restored_from"`
	NewRevision  int              `json:"new_revision"`
	Doc          *models.Document `json:"doc,omitempty"`
}

func (t *tools) restoreDocumentRevision(ctx context.Context, _ *sdk.CallToolRequest, in restoreInput) (*sdk.CallToolResult, *restoreResult, error) {
	if in.Revision <= 0 {
		return nil, nil, errors.New("revision is required")
	}
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	snap, err := t.store.GetDocumentRevision(ctx, doc.ID, in.Revision)
	if err != nil {
		return nil, nil, revisionErr(err, in.Revision)
	}
	// Restore the snapshot's content onto the live document; the write bumps the
	// revision and records a new "restore" audit entry.
	doc.Title = snap.Title
	doc.Status = snap.Status
	doc.Meta = snap.Meta
	doc.Body = snap.Body
	if err := t.saveDoc(ctx, doc, live.Event{Type: "documents", ID: doc.ID, Op: "restore_document_revision"}); err != nil {
		return nil, nil, err
	}
	out := &restoreResult{docSummary: summarize(doc), RestoredFrom: in.Revision, NewRevision: doc.Revision}
	if in.ReturnDoc {
		out.Doc = doc
	}
	return nil, out, nil
}

// revisionErr maps a missing revision to a friendly message.
func revisionErr(err error, revision int) error {
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("revision %d not found in this document's history", revision)
	}
	return err
}

// flattenNodes walks a document body and returns id → canonical JSON of each
// block/task's own scalar fields, EXCLUDING its "children" and "tasks" arrays so
// a parent is not reported as modified merely because a descendant changed.
// Tasks are flattened as their own nodes. Nodes without an id are skipped.
func flattenNodes(body map[string]any) map[string]string {
	out := map[string]string{}
	var walk func(blocks []any)
	walk = func(blocks []any) {
		for _, raw := range blocks {
			b, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if id, _ := b["id"].(string); id != "" {
				out[id] = ownFieldsJSON(b)
			}
			if tasks, ok := b["tasks"].([]any); ok {
				for _, traw := range tasks {
					if tm, ok := traw.(map[string]any); ok {
						if id, _ := tm["id"].(string); id != "" {
							out[id] = ownFieldsJSON(tm)
						}
					}
				}
			}
			if children, ok := b["children"].([]any); ok {
				walk(children)
			}
		}
	}
	sections, _ := body["sections"].([]any)
	walk(sections)
	return out
}

// ownFieldsJSON marshals a node's own fields (all keys except the nested
// "children" and "tasks" arrays) to canonical JSON. json.Marshal sorts map
// keys, so the output is stable for equality comparison.
func ownFieldsJSON(node map[string]any) string {
	own := make(map[string]any, len(node))
	for k, v := range node {
		if k == "children" || k == "tasks" {
			continue
		}
		own[k] = v
	}
	b, _ := json.Marshal(own)
	return string(b)
}
