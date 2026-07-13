package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/docmeta"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// --- push_document ----------------------------------------------------

type pushInput struct {
	Meta map[string]any `json:"meta" jsonschema:"document meta object; must include id, title, type"`
	Body map[string]any `json:"body" jsonschema:"document body — typically an object with a sections array"`
}

func (t *tools) pushDocument(ctx context.Context, _ *sdk.CallToolRequest, in pushInput) (*sdk.CallToolResult, *models.Document, error) {
	if in.Meta == nil {
		return nil, nil, errors.New("meta is required")
	}
	if in.Body == nil {
		in.Body = map[string]any{}
	}
	if err := validateBody(in.Body); err != nil {
		return nil, nil, err
	}
	id, _ := in.Meta["id"].(string)
	if id == "" {
		return nil, nil, errors.New("meta.id is required for push_document")
	}

	doc, err := docmeta.FromMeta(in.Meta, in.Body)
	if err != nil {
		return nil, nil, err
	}
	if doc.Title == "" {
		return nil, nil, errors.New("meta.title is required")
	}
	if doc.Type == "" {
		return nil, nil, errors.New("meta.type is required")
	}
	doc.ID = id

	existing, err := t.store.GetDocument(ctx, id)
	switch {
	case err == nil:
		// Preserve immutable fields from the existing row.
		doc.CreatedAt = existing.CreatedAt
		if err := t.store.UpdateDocument(ctx, doc); err != nil {
			return nil, nil, translateStoreErr(err)
		}
	case errors.Is(err, store.ErrNotFound):
		if err := t.store.CreateDocument(ctx, doc); err != nil {
			return nil, nil, translateStoreErr(err)
		}
	default:
		return nil, nil, translateStoreErr(err)
	}
	t.enqueue("documents", doc.ID)
	return nil, doc, nil
}

// --- list_documents ---------------------------------------------------

type listInput struct {
	Project string `json:"project,omitempty" jsonschema:"filter by project slug"`
	Type    string `json:"type,omitempty" jsonschema:"filter by document type (plan, spec, adr, report, brainstorm, journal)"`
	Status  string `json:"status,omitempty" jsonschema:"filter by status (todo, in-progress, complete, blocked, archived)"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max results; default 50, max 200"`
}

type listOutput struct {
	Items []docSummary `json:"items"`
}

func (t *tools) listDocuments(ctx context.Context, _ *sdk.CallToolRequest, in listInput) (*sdk.CallToolResult, *listOutput, error) {
	f := store.Filter{Limit: clampLimit(in.Limit, 50, 200)}
	if in.Project != "" {
		f.Project = &in.Project
	}
	if in.Type != "" {
		f.Type = &in.Type
	}
	if in.Status != "" {
		f.Status = &in.Status
	}
	docs, err := t.store.ListDocuments(ctx, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	out := &listOutput{Items: make([]docSummary, 0, len(docs))}
	for _, d := range docs {
		out.Items = append(out.Items, summarize(d))
	}
	return nil, out, nil
}

// --- get_document -----------------------------------------------------

type getInput struct {
	ID string `json:"id" jsonschema:"document id"`
}

func (t *tools) getDocument(ctx context.Context, _ *sdk.CallToolRequest, in getInput) (*sdk.CallToolResult, *models.Document, error) {
	doc, err := t.loadDoc(ctx, in.ID)
	if err != nil {
		return nil, nil, err
	}
	return nil, doc, nil
}

// --- search_documents -------------------------------------------------

type searchInput struct {
	Q       string `json:"q" jsonschema:"full-text query"`
	Project string `json:"project,omitempty" jsonschema:"limit to a project slug"`
	Type    string `json:"type,omitempty" jsonschema:"limit to a document type"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max results; default 20, max 100"`
}

func (t *tools) searchDocuments(ctx context.Context, _ *sdk.CallToolRequest, in searchInput) (*sdk.CallToolResult, *listOutput, error) {
	if in.Q == "" {
		return nil, nil, errors.New("q is required")
	}
	f := store.Filter{Limit: clampLimit(in.Limit, 20, 100)}
	if in.Project != "" {
		f.Project = &in.Project
	}
	if in.Type != "" {
		f.Type = &in.Type
	}
	docs, err := t.store.SearchDocuments(ctx, in.Q, f)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	out := &listOutput{Items: make([]docSummary, 0, len(docs))}
	for _, d := range docs {
		out.Items = append(out.Items, summarize(d))
	}
	return nil, out, nil
}

// --- archive_document -------------------------------------------------

type archiveInput struct {
	ID string `json:"id" jsonschema:"document id"`
}

func (t *tools) archiveDocument(ctx context.Context, _ *sdk.CallToolRequest, in archiveInput) (*sdk.CallToolResult, *okResult, error) {
	if in.ID == "" {
		return nil, nil, errors.New("id is required")
	}
	if err := t.store.ArchiveDocument(ctx, in.ID); err != nil {
		return nil, nil, translateStoreErr(err)
	}
	return nil, &okResult{OK: true}, nil
}

// --- update_document_meta --------------------------------------------

type updateMetaInput struct {
	ID   string         `json:"id" jsonschema:"document id"`
	Meta map[string]any `json:"meta" jsonschema:"new meta object — replaces the existing meta in full; known keys are re-promoted to typed columns"`
}

func (t *tools) updateDocumentMeta(ctx context.Context, _ *sdk.CallToolRequest, in updateMetaInput) (*sdk.CallToolResult, *models.Document, error) {
	if in.Meta == nil {
		return nil, nil, errors.New("meta is required")
	}
	doc, err := t.loadDoc(ctx, in.ID)
	if err != nil {
		return nil, nil, err
	}
	merged, err := docmeta.FromMeta(in.Meta, doc.Body)
	if err != nil {
		return nil, nil, err
	}
	if merged.Title == "" {
		return nil, nil, fmt.Errorf("meta.title is required")
	}
	if merged.Type == "" {
		return nil, nil, fmt.Errorf("meta.type is required")
	}
	docmeta.ApplyTo(doc, merged)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	return nil, doc, nil
}

// clampLimit returns n bounded by [1, max], substituting def when n<=0.
func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
