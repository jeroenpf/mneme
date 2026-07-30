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
	"github.com/jeroenpf/mneme/internal/relations"
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
		tools: &tools{store: st, enq: enq, bc: bc, client: client, cmd: command.NewDocuments(st, enq, bc), rel: &relations.Service{Store: st}},
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
	rel    *relations.Service // typed links + enriched related view
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

// register installs every surface-v2 tool on the SDK server. Adding a
// new tool means: define request/response structs, write the handler
// method on *tools, and add one addTool line here. Descriptions stay one
// sentence — cross-tool conventions live in the instructions const, paid
// for once per session instead of per tool.
func (t *tools) register(s *sdk.Server) {
	addTool(s, &sdk.Tool{
		Name:        "push_document",
		Description: "Create or fully rewrite a document by meta.id (meta keys and body block shapes are in the server instructions); prefer the surgical block/task tools for edits.",
	}, t.pushDocument)

	addTool(s, &sdk.Tool{
		Name:        "set_memory",
		Description: "Upsert a memory key at a scope (global | project | area); an empty value deletes the key.",
	}, t.setMemory)

	addTool(s, &sdk.Tool{
		Name:        "log_decision",
		Description: "Record a decision (title, decision, rationale, alternatives, consequences); pass id to update. Returns {public_id}.",
	}, t.logDecision)

	addTool(s, &sdk.Tool{
		Name:        "append_journal",
		Description: "Append a dev-journal entry (summary, accomplished, deferred); pass id to refine this session's entry. Returns {public_id}.",
	}, t.appendJournal)

	addTool(s, &sdk.Tool{
		Name:        "get_context_bundle",
		Description: "Session-start digest: memory, active plan, recent decisions and journal. Call this first.",
	}, t.getContextBundle)

	addTool(s, &sdk.Tool{
		Name:        "list_documents",
		Description: "List documents (no bodies), filtered by project, type, or status.",
	}, t.listDocuments)

	addTool(s, &sdk.Tool{
		Name:        "get_document",
		Description: "Fetch one document including its body.",
	}, t.getDocument)

	addTool(s, &sdk.Tool{
		Name:        "search",
		Description: "One ranked query across all content types; websearch syntax (phrases, OR, -exclusion).",
	}, t.search)

	addTool(s, &sdk.Tool{
		Name:        "resolve_reference",
		Description: "Resolve a mneme:// reference or public id to its entity plus the ids the surgical tools need.",
	}, t.resolveReference)

	addTool(s, &sdk.Tool{
		Name:        "tick_task",
		Description: "Toggle a task's done flag.",
	}, t.tickTask)

	addTool(s, &sdk.Tool{
		Name:        "update_task",
		Description: "Patch a task's title, content, done, or tags.",
	}, t.updateTask)

	addTool(s, &sdk.Tool{
		Name:        "add_task",
		Description: "Append a task to a subphase or task-list block (after_task_id positions it).",
	}, t.addTask)

	addTool(s, &sdk.Tool{
		Name:        "remove_task",
		Description: "Remove a task by id.",
	}, t.removeTask)

	addTool(s, &sdk.Tool{
		Name:        "update_section",
		Description: "Patch a section block's title/content by id.",
	}, t.updateSection)

	addTool(s, &sdk.Tool{
		Name:        "add_section",
		Description: "Append a validated block (section, code, diagram, table, callout, key-value, task-list, subphase); shapes are in the server instructions.",
	}, t.addSection)

	addTool(s, &sdk.Tool{
		Name:        "remove_section",
		Description: "Remove a block from body.sections by id.",
	}, t.removeSection)

	addTool(s, &sdk.Tool{
		Name:        "advance_phase",
		Description: "Complete the current plan phase and start the next.",
	}, t.advancePhase)

	addTool(s, &sdk.Tool{
		Name:        "archive_document",
		Description: "Set a document's status to archived.",
	}, t.archiveDocument)

	addTool(s, &sdk.Tool{
		Name:        "update_document_meta",
		Description: "Replace a document's meta, body untouched — send every key you want to keep.",
	}, t.updateDocumentMeta)

	addTool(s, &sdk.Tool{
		Name:        "link",
		Description: "Add a typed directed relation between two entities: relates-to, implements, supersedes, depends-on, or blocks.",
	}, t.link)

	addTool(s, &sdk.Tool{
		Name:        "unlink",
		Description: "Remove typed relations between two entities (one rel_type, or all when omitted).",
	}, t.unlink)

	addTool(s, &sdk.Tool{
		Name:        "get_related",
		Description: "Typed links, mentions, and backlinks for an entity.",
	}, t.getRelated)
}
