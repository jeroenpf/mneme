package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/ids"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// resolveRefInput is resolve_reference's single argument: a pasted mneme://
// reference or a bare public id.
type resolveRefInput struct {
	Ref string `json:"ref" jsonschema:"a mneme:// reference or bare public id to resolve, e.g. mneme://document/doc_… or mneme://document/doc_…/task/task_…"`
}

// resolveRefResult is the entity a reference resolves to, plus the stable ids a
// follow-up surgical tool needs. For a block or task, Document is the owning
// document summary — its id is the doc_id argument tick_task, update_section,
// etc. expect, and TargetID is the block/task id. Content is the typed entity
// (a whole document, a block/task node, or a decision/journal/snippet/solution/
// project record).
type resolveRefResult struct {
	Kind      string      `json:"kind"`
	Reference string      `json:"reference"`
	TargetID  string      `json:"target_id"`
	Document  *docSummary `json:"document,omitempty"`
	Content   any         `json:"content"`
}

func (t *tools) resolveReference(ctx context.Context, _ *sdk.CallToolRequest, in resolveRefInput) (*sdk.CallToolResult, *resolveRefResult, error) {
	ref, err := parseUserRef(in.Ref)
	if err != nil {
		return nil, nil, err
	}

	switch ref.Kind {
	case ids.KindDocument:
		doc, err := t.store.GetDocumentByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindDocument, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindDocument, doc.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindDocument), Reference: canonical, TargetID: doc.PublicID, Content: doc}, nil

	case ids.KindBlock, ids.KindTask:
		return t.resolveNested(ctx, ref)

	case ids.KindDecision:
		d, err := t.store.GetDecisionByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindDecision, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindDecision, d.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindDecision), Reference: canonical, TargetID: d.PublicID, Content: d}, nil

	case ids.KindJournal:
		e, err := t.store.GetJournalEntryByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindJournal, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindJournal, e.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindJournal), Reference: canonical, TargetID: e.PublicID, Content: e}, nil

	case ids.KindSnippet:
		sn, err := t.store.GetSnippetByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindSnippet, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindSnippet, sn.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindSnippet), Reference: canonical, TargetID: sn.PublicID, Content: sn}, nil

	case ids.KindSolution:
		sol, err := t.store.GetSolutionByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindSolution, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindSolution, sol.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindSolution), Reference: canonical, TargetID: sol.PublicID, Content: sol}, nil

	case ids.KindProject:
		p, err := t.store.GetProjectByPublicID(ctx, ref.ID)
		if err != nil {
			return nil, nil, refNotFound(err, ids.KindProject, ref.ID)
		}
		canonical, _ := ids.Ref(ids.KindProject, p.PublicID)
		return nil, &resolveRefResult{Kind: string(ids.KindProject), Reference: canonical, TargetID: p.PublicID, Content: p}, nil

	default:
		return nil, nil, fmt.Errorf("cannot resolve %s references", ref.Kind)
	}
}

// resolveNested resolves a block or task reference: fetch the owning document by
// its public id, then walk the body tree for the child id.
func (t *tools) resolveNested(ctx context.Context, ref ids.Reference) (*sdk.CallToolResult, *resolveRefResult, error) {
	doc, err := t.store.GetDocumentByPublicID(ctx, ref.DocID)
	if err != nil {
		return nil, nil, refNotFound(err, ids.KindDocument, ref.DocID)
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}

	var node map[string]any
	var canonical string
	switch ref.Kind {
	case ids.KindBlock:
		_, _, node = walkSectionsByID(sections, ref.ID)
		canonical, _ = ids.RefBlock(doc.PublicID, ref.ID)
	case ids.KindTask:
		_, _, node = walkTaskByID(sections, ref.ID)
		canonical, _ = ids.RefTask(doc.PublicID, ref.ID)
	}
	if node == nil {
		return nil, nil, fmt.Errorf("no %s %s in document %s", ref.Kind, ref.ID, doc.PublicID)
	}

	summary := summarize(doc)
	return nil, &resolveRefResult{
		Kind:      string(ref.Kind),
		Reference: canonical,
		TargetID:  ref.ID,
		Document:  &summary,
		Content:   node,
	}, nil
}

// parseUserRef parses what a user pasted: a canonical mneme:// reference, or a
// bare top-level public id (its prefix names the kind). A bare block/task id is
// rejected — those need their owning document to resolve.
func parseUserRef(s string) (ids.Reference, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ids.Reference{}, errors.New("ref is required")
	}
	if strings.HasPrefix(s, "mneme://") {
		return ids.ParseRef(s)
	}
	kind, _, err := ids.Parse(s)
	if err != nil {
		return ids.Reference{}, fmt.Errorf("%q is not a mneme:// reference or a public id", s)
	}
	if kind == ids.KindBlock || kind == ids.KindTask {
		return ids.Reference{}, fmt.Errorf("a bare %s id needs its owning document; paste the full reference, e.g. mneme://document/doc_…/%s/%s", kind, kind, s)
	}
	return ids.Reference{Kind: kind, ID: s}, nil
}

// refNotFound turns a store ErrNotFound into an actionable, reference-shaped
// message; other errors pass through unchanged.
func refNotFound(err error, kind ids.Kind, id string) error {
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no %s found for %s", kind, id)
	}
	return err
}
