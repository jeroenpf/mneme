// Package mcp wires the official MCP Go SDK to Mneme's data layer.
// It builds an mcp.Server with every tool Claude Code needs to push,
// query, and surgically edit documents, then exposes it over HTTP via
// the streamable transport.
//
// The SDK is imported as `sdk` because its package name (`mcp`) would
// otherwise collide with our own.
package mcp

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/command"
	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/store"
)

// Implementation identifies this server in MCP handshakes.
var implementation = &sdk.Implementation{
	Name:    "mneme",
	Version: "1.5.0",
}

// instructions is surfaced to MCP clients (Claude Code et al.) on connect.
// Tells the LLM what Mneme is for and which tools to reach for first.
// Kept terse on purpose — every token here lives in the client's context.
const instructions = `Mneme is the source of truth for plans, specs, and ongoing project knowledge that evolves between sessions.

When editing a plan or spec, prefer structured tools (tick_task, update_task, add_task, update_section, add_section, advance_phase) over push_document. They address blocks by stable ID, are server-validated, and cost ~100× fewer tokens than re-emitting the document.

Typical workflow:
1. list_documents (optionally filtered by project/type) to discover what exists.
2. search_documents (websearch syntax: phrases, OR, -exclusion) before assuming something is missing.
3. get_document only when you need the body.
4. Mutate with the narrowest tool that fits the change.

Use search(q, types?) for a single ranked query across every content type (documents, decisions, snippets, solutions, journal) instead of the per-type search tools when you don't know where an answer lives.

Whenever the user pastes a mneme:// reference or a bare public id (doc_…, dec_…, snip_…, etc.), call resolve_reference to fetch the target — it returns the typed entity plus the ids surgical tools need (a block/task's owning document.id and target_id). Never guess what a reference points to.

push_document is upsert-by-meta.id — reserve it for new documents or full rewrites. A document's project must already exist; call create_project(slug, name) once to register a new project before pushing documents that reference it.

At the start of every session, call get_context_bundle(project, area?) — one call returns merged memory, the active plan's status, recent decisions, relevant snippets, and recent journal entries as a paste-ready markdown digest. Use the individual read tools only when you need more detail or history than the bundle contains.

Use set_memory to record durable facts worth carrying across sessions. Use get_memory when you need to inspect a specific scope beyond the session bundle.

Record durable decisions with log_decision as you make them (tech/library choices, pattern selections, trade-off resolutions) so the "why" stays searchable via query_decisions. This is the mutable decision log — distinct from hardened ADRs that graduate to the repo as markdown.

Save reusable code patterns and project conventions with save_snippet as you establish them, and consult get_snippets / search_snippets before re-implementing one so the codebase stays consistent.

At the end of a work session, record what happened with append_journal (summary + what you accomplished + what you deferred). Use get_journal when you need history beyond the recent entries in the session bundle.

Before debugging a non-obvious or environment-specific error, call find_solution(query) to check for a known fix; after solving one, record it with log_solution (error_description + solution) so the next session finds it instead of re-debugging.

Repo-tracked files (CLAUDE.md, ADRs, READMEs, docs/specs/*.md) are not in Mneme; read those from disk.`

// Server holds the SDK Server plus the dependencies its tool handlers
// close over. It's safe to share across requests — the SDK manages
// per-session state inside its own machinery.
type Server struct {
	sdk   *sdk.Server
	tools *tools
}

// New constructs a Server with every Phase 1.4 tool registered. enq is the
// embedding job queue write tools notify after a successful write; pass nil
// (or a nil embed.Client-backed setup) to disable embedding — it defaults to
// a NopEnqueuer so the tools stay agnostic to whether Voyage is configured.
// bc is the live pub/sub broadcaster write tools notify after a successful
// write; pass nil to disable live updates — it defaults to a NopBroadcaster.
// client (may be nil) embeds the `search` query for hybrid ranking; nil ⇒
// FTS-only.
func New(st store.Store, enq embed.Enqueuer, bc live.Broadcaster, client embed.Client) *Server {
	if enq == nil {
		enq = embed.NopEnqueuer{}
	}
	if bc == nil {
		bc = live.NopBroadcaster{}
	}
	s := &Server{
		sdk:   sdk.NewServer(implementation, &sdk.ServerOptions{Instructions: instructions}),
		tools: &tools{store: st, enq: enq, bc: bc, client: client, cmd: command.NewDocuments(st, enq, bc)},
	}
	s.tools.register(s.sdk)
	return s
}

// Handler returns an http.Handler that serves the streamable HTTP
// transport. Mount it on a chi router at /mcp.
func (s *Server) Handler() http.Handler {
	return sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return s.sdk },
		nil,
	)
}

// tools bundles the store dependency so the per-tool handler files
// (tools_*.go) can stay focused on input/output shapes. enq receives an
// embedding job after each successful write (a NopEnqueuer when embedding
// is disabled). bc receives a live event after each successful write (a
// NopBroadcaster when live updates are disabled). client (may be nil)
// embeds the `search` query for hybrid ranking.
type tools struct {
	store  store.Store
	enq    embed.Enqueuer
	bc     live.Broadcaster
	client embed.Client
	cmd    *command.Documents // the single validated document write path
}

// addTool keeps the advertised output contract deliberately minimal. The
// concrete Out type still gives handlers compile-time type safety and the SDK
// still serializes it into content + structuredContent, but avoiding the
// inferred field-by-field schema prevents every tool from advertising a large
// shape callers do not need in order to invoke it. Input schemas stay fully
// inferred and validated by the SDK.
func addTool[In, Out any](s *sdk.Server, tool *sdk.Tool, handler sdk.ToolHandlerFor[In, Out]) {
	tool.OutputSchema = map[string]any{"type": "object"}
	sdk.AddTool(s, tool, handler)
}

// register installs every Phase 1.4 tool on the SDK server. Adding a
// new tool means: define request/response structs, write the handler
// method on *tools, and add one addTool line here.
func (t *tools) register(s *sdk.Server) {
	addTool(s, &sdk.Tool{
		Name:        "push_document",
		Description: "Create or upsert a document by meta.id. Validates block types and prose content (body prose allows blank-line paragraphs; lists/headings/code fences must be child blocks). Mints stable ids for blocks/tasks that omit them and rejects duplicate ids (listed in 'created'). Returns a compact summary (no body); pass return_doc:true for the full stored document.",
	}, t.pushDocument)

	addTool(s, &sdk.Tool{
		Name:        "create_project",
		Description: "Register a new project (slug + human-friendly name, optional description) so documents can reference it. push_document errors on an unknown project; create it first. Returns the stored project (slug normalized to kebab-case).",
	}, t.createProject)

	addTool(s, &sdk.Tool{
		Name:        "get_memory",
		Description: "Load persistent memory for a specific scope beyond the session bundle. Global, project, and area values are merged from least to most specific.",
	}, t.getMemory)

	addTool(s, &sdk.Tool{
		Name:        "set_memory",
		Description: "Upsert a memory key/value at a scope (global | project | area). project required for project/area scope; area required for area scope. Returns the stored entry.",
	}, t.setMemory)

	addTool(s, &sdk.Tool{
		Name:        "delete_memory",
		Description: "Delete a memory key at a scope. Same scope args as set_memory.",
	}, t.deleteMemory)

	addTool(s, &sdk.Tool{
		Name:        "get_env",
		Description: "Load a project's non-secret env registry as a flat {key: value} object — ports, service names, local URLs, Docker service names. Call this instead of asking \"what port does X run on?\". Never holds secrets.",
	}, t.getEnv)

	addTool(s, &sdk.Tool{
		Name:        "set_env",
		Description: "Upsert a non-secret env entry for a project (key + value, optional description) — ports, service names, local URLs. NEVER secrets/tokens/passwords. Returns the stored entry.",
	}, t.setEnv)

	addTool(s, &sdk.Tool{
		Name:        "list_env",
		Description: "List a project's env entries as full records including descriptions. Use get_env for a flat key/value map.",
	}, t.listEnv)

	addTool(s, &sdk.Tool{
		Name:        "log_decision",
		Description: "Record an architecture decision (ADR) — the mutable decision log Claude Code writes as a session side-effect. Omit id to create (title + decision required; project optional, omit for a global decision; status defaults to accepted). Pass id to update an existing decision, e.g. flip status proposed→accepted→deprecated. Returns the stored decision.",
	}, t.logDecision)

	addTool(s, &sdk.Tool{
		Name:        "get_decisions",
		Description: "List decisions newest-first, optionally filtered by project and/or status. Returns full records (rationale, alternatives, consequences) (default 20, max 100).",
	}, t.getDecisions)

	addTool(s, &sdk.Tool{
		Name:        "query_decisions",
		Description: "Full-text search decisions ranked by relevance — answers \"why did we choose X?\". Searches title, decision, rationale, alternatives, consequences. Optional project scope (default 10, max 50).",
	}, t.queryDecisions)

	addTool(s, &sdk.Tool{
		Name:        "save_snippet",
		Description: "Save a reusable code pattern or project convention — the snippet library that keeps Claude Code consistent without re-explaining. Omit id to create (title + content required; project optional, omit for a global snippet; language free-text like go/typescript/sql). Pass id to update an existing snippet (refine the pattern). Returns the stored snippet.",
	}, t.saveSnippet)

	addTool(s, &sdk.Tool{
		Name:        "get_snippets",
		Description: "List snippets newest-first, optionally filtered by project, language, and/or tag. Returns full records including content (default 20, max 100).",
	}, t.getSnippets)

	addTool(s, &sdk.Tool{
		Name:        "search_snippets",
		Description: "Full-text search snippets ranked by relevance — answers \"how do we do X in this project?\". Searches title, description, content. Optional project/language/tag filters (default 10, max 50).",
	}, t.searchSnippets)

	addTool(s, &sdk.Tool{
		Name:        "append_journal",
		Description: "Append a dev-journal entry — the per-session log of what was built, deferred, and changed. Omit id to create (summary required; project optional, omit for a global entry; session_ref is a free-text phase/session id). Pass id to refine the current session's entry as you go. Returns the stored entry.",
	}, t.appendJournal)

	addTool(s, &sdk.Tool{
		Name:        "get_journal",
		Description: "List dev-journal entries newest-first, optionally filtered by project and/or a since date (YYYY-MM-DD or RFC3339). Use limit for just the most recent few. Returns full entries (summary, accomplished, deferred) (default 20, max 100).",
	}, t.getJournal)

	addTool(s, &sdk.Tool{
		Name:        "log_solution",
		Description: "Log an error and the fix that worked — the searchable error/solution database. Omit id to create (error_description + solution required; project optional, omit for a global gotcha). Pass id to refine an existing entry. Returns the stored solution.",
	}, t.logSolution)

	addTool(s, &sdk.Tool{
		Name:        "find_solution",
		Description: "Search the error/solution database for a fix, ranked by relevance — call this BEFORE debugging to check whether an error has a known fix. Returns the top 3 matches by default (pass limit to widen). Optional project/tag filters.",
	}, t.findSolution)

	addTool(s, &sdk.Tool{
		Name:        "get_context_bundle",
		Description: "Return the paste-ready markdown digest for starting a project session: merged memory, active-plan status, recent decisions, relevant snippets, and recent journal entries. Call this first in every session.",
	}, t.getContextBundle)

	addTool(s, &sdk.Tool{
		Name:        "list_documents",
		Description: "List documents, optionally filtered by project, type, or status. Body is omitted for compactness.",
	}, t.listDocuments)

	addTool(s, &sdk.Tool{
		Name:        "get_document",
		Description: "Fetch a single document including its body.",
	}, t.getDocument)

	addTool(s, &sdk.Tool{
		Name:        "search_documents",
		Description: "Full-text search across documents. Returns ranked matches without bodies.",
	}, t.searchDocuments)

	addTool(s, &sdk.Tool{
		Name:        "search",
		Description: "Search documents, decisions, snippets, solutions, and journal in one ranked query when you do not know which content type holds the answer. Returns compact hits and is a superset of search_documents.",
	}, t.search)

	addTool(s, &sdk.Tool{
		Name:        "resolve_reference",
		Description: "Resolve a pasted mneme:// reference (or a bare public id like doc_…/dec_…) to its entity. Returns {kind, reference, target_id, document?, content}: the typed entity plus the ids a follow-up surgical tool needs — for a block or task, document.id is the doc_id argument for tick_task/update_task/update_section and target_id is the block/task id. Call this whenever the user pastes a mneme:// reference instead of guessing what it points to.",
	}, t.resolveReference)

	addTool(s, &sdk.Tool{
		Name:        "retry_failed_embeddings",
		Description: "Re-enqueue every source whose last embedding attempt failed terminally (the 'failed' count in search status). Use after fixing a transient outage. Returns how many sources were queued for retry.",
	}, t.retryFailedEmbeddings)

	addTool(s, &sdk.Tool{
		Name:        "tick_task",
		Description: "Toggle a task's done flag. Returns {task_id, done}; pass return_doc for the full updated document.",
	}, t.tickTask)

	addTool(s, &sdk.Tool{
		Name:        "update_task",
		Description: "Patch a task's title, content, done, or tags. Returns the updated task; pass return_doc for the full document.",
	}, t.updateTask)

	addTool(s, &sdk.Tool{
		Name:        "add_task",
		Description: "Append a task to a subphase or task-list block (after_task_id to position it). Generates the task id when omitted, and rejects a supplied id already used in the document. Returns the new task; pass return_doc for the full document.",
	}, t.addTask)

	addTool(s, &sdk.Tool{
		Name:        "remove_task",
		Description: "Remove a task from its subphase.",
	}, t.removeTask)

	addTool(s, &sdk.Tool{
		Name:        "update_section",
		Description: "Patch a section block's fields. Returns the patched block; pass return_doc for the full document.",
	}, t.updateSection)

	addTool(s, &sdk.Tool{
		Name:        "add_section",
		Description: "Append any validated document block, including a section, code, Mermaid diagram, table, callout, key-value, task-list, or subphase. Exact shapes are documented by push_document.body. Mints ids for the block and any nested children/tasks that omit them (listed in 'created'). Returns the new block; pass return_doc for the full document.",
	}, t.addSection)

	addTool(s, &sdk.Tool{
		Name:        "remove_section",
		Description: "Remove a section block from body.sections.",
	}, t.removeSection)

	addTool(s, &sdk.Tool{
		Name:        "advance_phase",
		Description: "Flip the current meta.phases entry wip->done and the next todo->wip. Returns {completed_index, next_index, phase_current, phase_total, status}; pass return_doc for the full document. Falls back to bumping phase_current when meta.phases[] is absent.",
	}, t.advancePhase)

	addTool(s, &sdk.Tool{
		Name:        "archive_document",
		Description: "Set a document's status to archived.",
	}, t.archiveDocument)

	addTool(s, &sdk.Tool{
		Name:        "update_document_meta",
		Description: "Replace a document's meta object (body untouched). Returns a compact summary; pass return_doc:true for the full document.",
	}, t.updateDocumentMeta)

	addTool(s, &sdk.Tool{
		Name:        "get_document_history",
		Description: "List a document's revision history newest-first: each entry's revision, operation, actor, affected ids, title, status, and timestamp (no body). Use diff_document_revisions or restore_document_revision to work with a revision's content.",
	}, t.getDocumentHistory)

	addTool(s, &sdk.Tool{
		Name:        "diff_document_revisions",
		Description: "Report what changed between two revisions of a document without re-emitting them: the node ids added, removed, and modified, plus whether title or status changed. Pass from_revision (required) and to_revision (omit to compare against the current document).",
	}, t.diffDocumentRevisions)

	addTool(s, &sdk.Tool{
		Name:        "restore_document_revision",
		Description: "Restore a past revision's content (title/status/meta/body) onto the document. History is forward-only: this writes a NEW revision rather than rewriting the past. Returns the restored summary and new revision; pass return_doc for the full document.",
	}, t.restoreDocumentRevision)

	addTool(s, &sdk.Tool{
		Name:        "lint_documents",
		Description: "Read-only sweep of every stored document (all projects and statuses, archived included) for inline-only violations and structural problems (unknown types/fields) that predate write-path validation. Each hit carries doc_id, block_id, path, field, what was found, and an excerpt — doc_id+block_id feed update_section/update_task fixes. Changes nothing.",
	}, t.lintDocuments)
}
