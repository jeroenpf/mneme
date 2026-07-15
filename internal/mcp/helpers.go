package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/live"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// okResult is the tiny structured output for "side-effect succeeded"
// tools that don't have a natural payload (remove_task, archive, etc.).
type okResult struct {
	OK bool `json:"ok"`
}

// docSummary is the lightweight view of a document used by
// list_documents and search_documents — body stripped.
type docSummary struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Project      *string  `json:"project,omitempty"`
	Type         string   `json:"type"`
	Status       string   `json:"status"`
	Ticket       *string  `json:"ticket,omitempty"`
	Tags         []string `json:"tags"`
	PhaseCurrent *int     `json:"phase_current,omitempty"`
	PhaseTotal   *int     `json:"phase_total,omitempty"`
}

func summarize(d *models.Document) docSummary {
	return docSummary{
		ID:           d.ID,
		Title:        d.Title,
		Project:      d.Project,
		Type:         d.Type,
		Status:       d.Status,
		Ticket:       d.Ticket,
		Tags:         d.Tags,
		PhaseCurrent: d.PhaseCurrent,
		PhaseTotal:   d.PhaseTotal,
	}
}

// docWriteResult is the lean confirmation for push_document /
// update_document_meta: the document's meta summary, no body. The full
// document is attached solely when the caller passes return_doc.
type docWriteResult struct {
	docSummary
	Doc *models.Document `json:"doc,omitempty"`
}

// writeResult builds a docWriteResult, attaching the full document only
// when returnDoc is set.
func writeResult(doc *models.Document, returnDoc bool) *docWriteResult {
	r := &docWriteResult{docSummary: summarize(doc)}
	if returnDoc {
		r.Doc = doc
	}
	return r
}

// translateStoreErr maps known store errors to messages aimed at the
// LLM caller. Returned errors become CallToolResult IsError content.
func translateStoreErr(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return errors.New("document not found")
	case errors.Is(err, store.ErrInvalidProject):
		return errors.New("unknown project — create it before pushing documents that reference it")
	case errors.Is(err, store.ErrDuplicateID):
		return errors.New("document id already exists")
	case errors.Is(err, store.ErrDuplicateProject):
		return errors.New("project already exists")
	default:
		return err
	}
}

// loadDoc fetches a document and pre-translates ErrNotFound — used by
// every R/M/W tool.
func (t *tools) loadDoc(ctx context.Context, id string) (*models.Document, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	doc, err := t.store.GetDocument(ctx, id)
	if err != nil {
		return nil, translateStoreErr(err)
	}
	return doc, nil
}

// saveDoc persists an updated document, translating store errors. On
// success it enqueues the document for re-embedding — every section/task/
// phase/meta edit tool routes through here, so this is the single point
// that keeps embeddings in step with document mutations.
func (t *tools) saveDoc(ctx context.Context, doc *models.Document) error {
	if err := t.store.UpdateDocument(ctx, doc); err != nil {
		return translateStoreErr(err)
	}
	t.enqueue("documents", doc.ID)
	return nil
}

// enqueue notifies the embedding worker that a source changed, and
// broadcasts the same change to live SSE subscribers. It's a no-op when
// embedding/live are disabled (NopEnqueuer/NopBroadcaster) and skips empty
// ids. Documents are deliberately excluded from the broadcast: their edit
// handlers broadcast themselves, naming the changed block (see P3), so a
// generic doc-level event here would duplicate that. Embedding is still
// enqueued for documents.
func (t *tools) enqueue(sourceType, id string) {
	if id == "" {
		return
	}
	t.enq.Enqueue(embed.SourceRef{Type: sourceType, ID: id})
	if sourceType != "documents" {
		t.bc.Broadcast(live.Event{Type: sourceType, ID: id})
	}
}

// broadcast is the direct live hook for writes that don't route through
// enqueue (archive, memory, env) — and, in P3, for document edits that
// carry a block id.
func (t *tools) broadcast(ev live.Event) { t.bc.Broadcast(ev) }

// sectionsArray returns body.sections as []any, creating it if missing.
// Mutations to the returned slice must be written back via setSections.
func sectionsArray(body map[string]any) ([]any, error) {
	if body == nil {
		return []any{}, nil
	}
	raw, ok := body["sections"]
	if !ok {
		return []any{}, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("body.sections must be an array")
	}
	return arr, nil
}

func setSections(body map[string]any, sections []any) {
	body["sections"] = sections
}

// walkSectionsByID finds the first block whose "id" matches id and
// returns its parent slice, its index within that slice, and the block
// itself. It descends into "children" arrays. Returns (nil, 0, nil) if
// no match.
func walkSectionsByID(blocks []any, id string) (parent []any, idx int, block map[string]any) {
	for i, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if bid, _ := b["id"].(string); bid == id {
			return blocks, i, b
		}
		if children, ok := b["children"].([]any); ok {
			if p, ix, found := walkSectionsByID(children, id); found != nil {
				return p, ix, found
			}
		}
	}
	return nil, 0, nil
}

// hasTasks reports whether a block owns a "tasks" array. The two plan
// block types that carry tasks are subphase and task-list; tick/update/
// remove/add_task should reach into either.
func hasTasks(b map[string]any) bool {
	t, _ := b["type"].(string)
	return t == "subphase" || t == "task-list"
}

// walkTaskByID scans every task-holding block reachable from blocks
// (subphase or task-list, including nested in section children) and
// finds a task whose "id" matches id. Returns the container block (whose
// "tasks" array owns the task), the index within that array, and the
// task map. Returns (nil, 0, nil) if no match.
func walkTaskByID(blocks []any, id string) (container map[string]any, idx int, task map[string]any) {
	for _, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if hasTasks(b) {
			tasks, _ := b["tasks"].([]any)
			for i, traw := range tasks {
				tm, ok := traw.(map[string]any)
				if !ok {
					continue
				}
				if tid, _ := tm["id"].(string); tid == id {
					return b, i, tm
				}
			}
		}
		if children, ok := b["children"].([]any); ok {
			if c, ix, tm := walkTaskByID(children, id); tm != nil {
				return c, ix, tm
			}
		}
	}
	return nil, 0, nil
}

// findTaskContainer finds a task-holding block (subphase or task-list) by
// id, anywhere in the tree.
func findTaskContainer(blocks []any, id string) map[string]any {
	for _, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if hasTasks(b) {
			if bid, _ := b["id"].(string); bid == id {
				return b
			}
		}
		if children, ok := b["children"].([]any); ok {
			if found := findTaskContainer(children, id); found != nil {
				return found
			}
		}
	}
	return nil
}
