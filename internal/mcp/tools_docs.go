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

// requireProjectForPlan rejects a plan with no project. A projectless
// plan is invisible to get_context_bundle and project-scoped
// list_documents (both filter by project), so it silently never becomes
// any project's active plan — the orphaned-plan trap. Other doc types
// may legitimately be global.
func requireProjectForPlan(doc *models.Document) error {
	if doc.Type == models.TypePlan && (doc.Project == nil || *doc.Project == "") {
		return errors.New("meta.project is required for a plan so it surfaces in get_context_bundle and project-scoped lists; register it first with create_project if it does not exist")
	}
	return nil
}

// --- push_document ----------------------------------------------------

type pushInput struct {
	Meta map[string]any `json:"meta" jsonschema:"document meta. Keys: id (required, stable slug), title (required), type (required; one of plan, report, spec, adr, brainstorm, journal), project (project slug; REQUIRED for type=plan so the plan surfaces in get_context_bundle — the project must already exist via create_project), status (one of todo, in-progress, complete, blocked, archived; default todo — this is the DOCUMENT lifecycle status, distinct from phase/task state which uses wip/done), category, ticket, repo, tags (array of strings), phase_current + phase_total (integers, plan progress). Unknown keys are stored verbatim."`
	Body map[string]any `json:"body" jsonschema:"document body as a block tree: {sections:[...]}. Every block needs a unique id and a type; unknown or misnamed fields are REJECTED (not silently dropped), so use these exact field names. Block shapes: section {title, content?, children?[]} — content is markdown prose under the heading, children are nested blocks; text {content}; task-list {title?, tasks:[{id, title, content?, done}]}; subphase {num, title, session?, description?, tasks[], children?[]}; callout {variant, title?, content} where variant is one of info|warn|success|danger|note; code {lang, filename?, content} where content is the code and lang is e.g. go|ts|sql|bash; diagram {title?, content} where content is mermaid source; table {title?, cols[], rows[][]}; key-value {title?, data{}}. INLINE-ONLY: every content/title/description string renders as inline markdown only (emphasis, code spans, links, :icon:). Blank lines, - / * bullet lists, 1. numbered lists, and # headings DO NOT render — they collapse to one line and the write path REJECTS them. For multiple paragraphs / lists / labelled fields, use child blocks: one text block per paragraph, callout for notes, key-value for labelled fields, table for tabular data. Exceptions: code.content and diagram.content keep their newlines."`
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
	if err := requireProjectForPlan(doc); err != nil {
		return nil, nil, err
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
	Meta map[string]any `json:"meta" jsonschema:"new meta object — REPLACES the existing meta in full, so send every key you want to keep (a dropped project would orphan a plan). Same keys and valid values as push_document: id, title, type (plan|report|spec|adr|brainstorm|journal), project (required for plans), status (todo|in-progress|complete|blocked|archived), category, ticket, repo, tags, phase_current, phase_total."`
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
	if err := requireProjectForPlan(merged); err != nil {
		return nil, nil, err
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
