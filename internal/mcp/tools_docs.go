package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/command"
	"github.com/jeroenpf/mneme/internal/docmeta"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/slug"
	"github.com/jeroenpf/mneme/internal/store"
)

// requireProjectForPlan rejects a plan with no project. A projectless
// plan is invisible to get_context_bundle and project-scoped
// list_documents (both filter by project), so it silently never becomes
// any project's active plan — the orphaned-plan trap. Other doc types
// may legitimately be global.
func requireProjectForPlan(doc *models.Document) error {
	if doc.Type == models.TypePlan && (doc.Project == nil || *doc.Project == "") {
		return errors.New("meta.project is required for a plan so it surfaces in get_context_bundle and project-scoped lists; pass project_create:true if the project does not exist yet")
	}
	return nil
}

// ensureProject verifies meta.project exists, creating it when create is
// set, else returning a teaching error that lists the known slugs.
func (t *tools) ensureProject(ctx context.Context, project string, create bool) error {
	_, err := t.store.GetProject(ctx, project)
	if err == nil {
		return nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return translateStoreErr(err)
	}
	if create {
		if norm := slug.Make(project); norm != project {
			return fmt.Errorf("project %q is not a valid slug — use %q as meta.project", project, norm)
		}
		if cerr := t.store.CreateProject(ctx, &models.Project{Slug: project, Name: project}); cerr != nil {
			return translateStoreErr(cerr)
		}
		return nil
	}
	stats, lerr := t.store.ListProjects(ctx)
	if lerr != nil {
		return translateStoreErr(lerr)
	}
	slugs := make([]string, 0, len(stats))
	for _, s := range stats {
		slugs = append(slugs, s.Slug)
	}
	return fmt.Errorf("unknown project %q — known projects: %s. Pass project_create:true to create it", project, strings.Join(slugs, ", "))
}

// --- push_document ----------------------------------------------------

type pushInput struct {
	Meta      map[string]any `json:"meta" jsonschema:"document meta. Keys: id (required, stable slug), title (required), type (required; one of plan, report, spec, adr, brainstorm, journal), project (project slug; REQUIRED for type=plan so the plan surfaces in get_context_bundle — the project must already exist via create_project), status (one of todo, in-progress, complete, blocked, archived; default todo — this is the DOCUMENT lifecycle status, distinct from phase/task state which uses wip/done), category, ticket, repo, tags (array of strings), phase_current + phase_total (integers, plan progress). Unknown keys are stored verbatim."`
	Body      map[string]any `json:"body" jsonschema:"document body as a block tree: {sections:[...]}. Every block needs a type; an id is optional — Mneme mints a stable one when you omit it, and every id (block and task) must be unique within the document (duplicates are rejected). The response's 'created' array lists any ids the server minted. Unknown or misnamed fields are REJECTED (not silently dropped), so use these exact field names. Block shapes: section {title, content?, children?[]} — content is markdown prose under the heading, children are nested blocks; text {content}; task-list {title?, tasks:[{id, title, content?, done}]}; subphase {num, title, session?, description?, tasks[], children?[]}; callout {variant, title?, content} where variant is one of info|warn|success|danger|note; code {lang, filename?, content} where content is the code and lang is e.g. go|ts|sql|bash; diagram {title?, content} where content is mermaid source; table {title?, cols[], rows[][]}; key-value {title?, data{}}. PROSE: content/title/description render inline markdown (emphasis, code spans, links, :icon:). Body prose — section/text/callout content and subphase description — MAY hold multiple paragraphs: separate them with a blank line. Titles and terse fields (task titles, key-value values) stay a single line. In every field, lists (- / * / 1.), # headings, and fenced code blocks DO NOT render and are REJECTED — use child blocks: task-list for a list, code for a code block, a nested section for a heading, key-value for labelled fields, table for tabular data. Exceptions: code.content and diagram.content keep their newlines verbatim."`
	ReturnDoc     bool `json:"return_doc,omitempty" jsonschema:"when true, also return the full stored document; default false (a compact summary is returned)"`
	ProjectCreate bool `json:"project_create,omitempty" jsonschema:"create meta.project if it does not exist yet"`
	// ExpectedRevision opts into optimistic concurrency on the update path:
	// when set, the push applies only if the stored document's revision still
	// equals it, else a revision-conflict error is returned. Omit for
	// last-writer-wins (and always for a first create).
	ExpectedRevision *int `json:"expected_revision,omitempty" jsonschema:"if set, update only when the stored document is still at this revision (optimistic concurrency); omit for last-writer-wins"`
}

func (t *tools) pushDocument(ctx context.Context, _ *sdk.CallToolRequest, in pushInput) (*sdk.CallToolResult, *docWriteResult, error) {
	if in.Meta == nil {
		return nil, nil, errors.New("meta is required")
	}
	if in.Body == nil {
		in.Body = map[string]any{}
	}
	if err := validateBody(in.Body); err != nil {
		return nil, nil, err
	}
	// Mint ids for blocks/tasks that omit one and reject any that are
	// non-string or duplicated, so nested identity is server-authoritative.
	created, err := normalizeBodyIDs(in.Body)
	if err != nil {
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
	if doc.Project != nil && *doc.Project != "" {
		if err := t.ensureProject(ctx, *doc.Project, in.ProjectCreate); err != nil {
			return nil, nil, err
		}
	}
	doc.ID = id

	// A whole-document push has no single edited block, so the live event and
	// audit record are document-level (empty BlockID → the affected id is the doc).
	ev := live.Event{Type: "documents", ID: doc.ID, Op: "push_document"}
	existing, err := t.store.GetDocument(ctx, id)
	switch {
	case err == nil:
		// Preserve immutable fields from the existing row.
		doc.CreatedAt = existing.CreatedAt
		if err := t.cmd.Update(ctx, doc, command.Write{Op: "push_document", Actor: "mcp", Event: ev, Expected: in.ExpectedRevision}); err != nil {
			return nil, nil, translateStoreErr(err)
		}
	case errors.Is(err, store.ErrNotFound):
		if err := t.cmd.Create(ctx, doc, command.Write{Op: "push_document", Actor: "mcp", Event: ev}); err != nil {
			return nil, nil, translateStoreErr(err)
		}
	default:
		return nil, nil, translateStoreErr(err)
	}
	res := writeResult(doc, in.ReturnDoc)
	res.Created = created
	return nil, res, nil
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

// --- archive_document -------------------------------------------------

type archiveInput struct {
	ID string `json:"id" jsonschema:"document id"`
}

func (t *tools) archiveDocument(ctx context.Context, _ *sdk.CallToolRequest, in archiveInput) (*sdk.CallToolResult, *okResult, error) {
	doc, err := t.loadDoc(ctx, in.ID)
	if err != nil {
		return nil, nil, err
	}
	// Already archived → safe idempotent no-op: skip the redundant revision.
	if doc.Status == models.StatusArchived {
		return nil, &okResult{OK: true}, nil
	}
	// Route through the command service so the archive records a revision
	// snapshot (op archive_document), re-embeds, and broadcasts — exactly once,
	// like every other document write (roadmap P6).
	doc.Status = models.StatusArchived
	if err := t.saveDoc(ctx, doc, live.Event{Type: "documents", ID: doc.ID, Op: "archive_document"}); err != nil {
		return nil, nil, err
	}
	return nil, &okResult{OK: true}, nil
}

// --- update_document_meta --------------------------------------------

type updateMetaInput struct {
	ID        string         `json:"id" jsonschema:"document id"`
	Meta      map[string]any `json:"meta" jsonschema:"new meta object — REPLACES the existing meta in full, so send every key you want to keep; dropping project would orphan a plan. Uses the same keys and rules as push_document.meta."`
	ReturnDoc bool           `json:"return_doc,omitempty" jsonschema:"when true, also return the full updated document; default false (a compact summary is returned)"`
}

func (t *tools) updateDocumentMeta(ctx context.Context, _ *sdk.CallToolRequest, in updateMetaInput) (*sdk.CallToolResult, *docWriteResult, error) {
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
	if err := t.saveDoc(ctx, doc, live.Event{Type: "documents", ID: doc.ID, Op: "update_document_meta"}); err != nil {
		return nil, nil, err
	}
	return nil, writeResult(doc, in.ReturnDoc), nil
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
