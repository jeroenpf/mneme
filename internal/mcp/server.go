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

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// Implementation identifies this server in MCP handshakes.
var implementation = &sdk.Implementation{
	Name:    "mneme",
	Version: "1.4.0",
}

// instructions is surfaced to MCP clients (Claude Code et al.) on connect.
// Tells the LLM what Mneme is for and which tools to reach for first.
// Kept terse on purpose — every token here lives in the client's context.
const instructions = `Mneme is the source of truth for plans, specs, and ongoing project knowledge — anything under ` + "`.architecture/`" + ` or that evolves between sessions.

When editing a plan or spec, prefer structured tools (tick_task, update_task, add_task, update_section, add_section, advance_phase) over push_document. They address blocks by stable ID, are server-validated, and cost ~100× fewer tokens than re-emitting the document.

Typical workflow:
1. list_documents (optionally filtered by project/type) to discover what exists.
2. search_documents (websearch syntax: phrases, OR, -exclusion) before assuming something is missing.
3. get_document only when you need the body.
4. Mutate with the narrowest tool that fits the change.

Use search(q, types?) for a single ranked query across every content type (documents, decisions, snippets, solutions, journal) instead of the per-type search tools when you don't know where an answer lives.

push_document is upsert-by-meta.id — reserve it for new documents or full rewrites. A document's project must already exist; call create_project(slug, name) once to register a new project before pushing documents that reference it.

At the START of every session, call get_context_bundle(project, area?) — one call returns merged memory, the active plan's status, recent decisions, relevant snippets, and recent journal entries, as both structured data and a paste-ready digest. Prefer it over calling get_memory / get_decisions / get_snippets / get_journal individually at startup.

At session start, call get_memory(scope) to load persistent context (global, or project/area for the active project) instead of asking the user to re-explain. Use set_memory to record durable facts worth carrying across sessions.

Record durable decisions with log_decision as you make them (tech/library choices, pattern selections, trade-off resolutions) so the "why" stays searchable via query_decisions. This is the mutable decision log — distinct from hardened ADRs that graduate to the repo as markdown.

Save reusable code patterns and project conventions with save_snippet as you establish them, and consult get_snippets / search_snippets before re-implementing one so the codebase stays consistent.

At the end of a work session, record what happened with append_journal (summary + what you accomplished + what you deferred); call get_journal at the start of the next session to orient on where you left off.

Before debugging a non-obvious or environment-specific error, call find_solution(query) to check for a known fix; after solving one, record it with log_solution (error_description + solution) so the next session finds it instead of re-debugging.

Repo-tracked files (CLAUDE.md, ADRs, READMEs, .architecture/specs/*.md) are not in Mneme; read those from disk.`

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
// client (may be nil) embeds the `search` query for hybrid ranking; nil ⇒
// FTS-only.
func New(st store.Store, enq embed.Enqueuer, client embed.Client) *Server {
	if enq == nil {
		enq = embed.NopEnqueuer{}
	}
	s := &Server{
		sdk:   sdk.NewServer(implementation, &sdk.ServerOptions{Instructions: instructions}),
		tools: &tools{store: st, enq: enq, client: client},
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
// is disabled). client (may be nil) embeds the `search` query for hybrid
// ranking.
type tools struct {
	store  store.Store
	enq    embed.Enqueuer
	client embed.Client
}

// register installs every Phase 1.4 tool on the SDK server. Adding a
// new tool means: define request/response structs, write the handler
// method on *tools, and add one sdk.AddTool line here.
func (t *tools) register(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name:        "push_document",
		Description: "Create or upsert a document by meta.id. Validates block types. Returns the stored document.",
	}, t.pushDocument)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "create_project",
		Description: "Register a new project (slug + human-friendly name, optional description) so documents can reference it. push_document errors on an unknown project; create it first. Returns the stored project (slug normalized to kebab-case).",
	}, t.createProject)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_memory",
		Description: "Load persistent memory for a scope as a flat key/value object. scope=global returns global keys; scope=project (with project) merges global+project; scope=area (with project+area) merges global+project+area, most-specific winning. Call at session start to orient.",
	}, t.getMemory)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "set_memory",
		Description: "Upsert a memory key/value at a scope (global | project | area). project required for project/area scope; area required for area scope. Returns the stored entry.",
	}, t.setMemory)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "delete_memory",
		Description: "Delete a memory key at a scope. Same scope args as set_memory.",
	}, t.deleteMemory)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_env",
		Description: "Load a project's non-secret env registry as a flat {key: value} object — ports, service names, local URLs, Docker service names. Call this instead of asking \"what port does X run on?\". Never holds secrets.",
	}, t.getEnv)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "set_env",
		Description: "Upsert a non-secret env entry for a project (key + value, optional description) — ports, service names, local URLs. NEVER secrets/tokens/passwords. Returns the stored entry.",
	}, t.setEnv)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_env",
		Description: "List a project's env entries as full records including descriptions. Use get_env for a flat key/value map.",
	}, t.listEnv)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "log_decision",
		Description: "Record an architecture decision (ADR) — the mutable decision log Claude Code writes as a session side-effect. Omit id to create (title + decision required; project optional, omit for a global decision; status defaults to accepted). Pass id to update an existing decision, e.g. flip status proposed→accepted→deprecated. Returns the stored decision.",
	}, t.logDecision)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_decisions",
		Description: "List decisions newest-first, optionally filtered by project and/or status. Returns full records (rationale, alternatives, consequences).",
	}, t.getDecisions)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "query_decisions",
		Description: "Full-text search decisions ranked by relevance — answers \"why did we choose X?\". Searches title, decision, rationale, alternatives, consequences. Optional project scope.",
	}, t.queryDecisions)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "save_snippet",
		Description: "Save a reusable code pattern or project convention — the snippet library that keeps Claude Code consistent without re-explaining. Omit id to create (title + content required; project optional, omit for a global snippet; language free-text like go/typescript/sql). Pass id to update an existing snippet (refine the pattern). Returns the stored snippet.",
	}, t.saveSnippet)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_snippets",
		Description: "List snippets newest-first, optionally filtered by project, language, and/or tag. Returns full records including content.",
	}, t.getSnippets)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "search_snippets",
		Description: "Full-text search snippets ranked by relevance — answers \"how do we do X in this project?\". Searches title, description, content. Optional project/language/tag filters.",
	}, t.searchSnippets)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "append_journal",
		Description: "Append a dev-journal entry — the per-session log of what was built, deferred, and changed. Omit id to create (summary required; project optional, omit for a global entry; session_ref is a free-text phase/session id). Pass id to refine the current session's entry as you go. Returns the stored entry.",
	}, t.appendJournal)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_journal",
		Description: "List dev-journal entries newest-first, optionally filtered by project and/or a since date (YYYY-MM-DD or RFC3339). Use limit for just the most recent few. Returns full entries (summary, accomplished, deferred).",
	}, t.getJournal)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "log_solution",
		Description: "Log an error and the fix that worked — the searchable error/solution database. Omit id to create (error_description + solution required; project optional, omit for a global gotcha). Pass id to refine an existing entry. Returns the stored solution.",
	}, t.logSolution)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "find_solution",
		Description: "Search the error/solution database for a fix, ranked by relevance — call this BEFORE debugging to check whether an error has a known fix. Returns the top 3 matches by default (pass limit to widen). Optional project/tag filters.",
	}, t.findSolution)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_context_bundle",
		Description: "Assemble everything needed to start a session on a project in one call: merged memory (global+project+area), the active plan's status, recent decisions, relevant snippets, and recent journal entries — returned as structured data plus a paste-ready markdown digest. Call this FIRST in every session. Args: project (required), area (optional).",
	}, t.getContextBundle)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_documents",
		Description: "List documents, optionally filtered by project, type, or status. Body is omitted for compactness.",
	}, t.listDocuments)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_document",
		Description: "Fetch a single document including its body.",
	}, t.getDocument)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "search_documents",
		Description: "Full-text search across documents. Returns ranked matches without bodies.",
	}, t.searchDocuments)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "search",
		Description: "Unified full-text search across documents, decisions, snippets, solutions, and journal — ranked by relevance, newest-first on ties. Args: q (required); types (optional subset of documents|decisions|snippets|solutions|journal, default all); project (optional scope); limit (default 10). Returns ranked hits with type, id, title, excerpt, project, score. Superset of search_documents.",
	}, t.search)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "tick_task",
		Description: "Toggle a task's done flag. Returns the updated task.",
	}, t.tickTask)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "update_task",
		Description: "Patch a task's title, content, done, or tags fields.",
	}, t.updateTask)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "add_task",
		Description: "Append a task to a subphase. If after_task_id is set, insert immediately after that task.",
	}, t.addTask)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "remove_task",
		Description: "Remove a task from its subphase.",
	}, t.removeTask)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "update_section",
		Description: "Patch any field on a section block (title, description, variant, content, etc.).",
	}, t.updateSection)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "add_section",
		Description: "Append a section to body.sections. If after_section_id is set, insert immediately after that section.",
	}, t.addSection)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "remove_section",
		Description: "Remove a section block from body.sections.",
	}, t.removeSection)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "advance_phase",
		Description: "Flip the current meta.phases entry from wip to done and the next one from todo to wip. Bumps the typed phase_current column.",
	}, t.advancePhase)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "archive_document",
		Description: "Set a document's status to archived.",
	}, t.archiveDocument)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "update_document_meta",
		Description: "Replace a document's meta object. Body is left untouched. Known meta keys are re-promoted to typed columns.",
	}, t.updateDocumentMeta)
}
