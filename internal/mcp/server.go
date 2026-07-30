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
	Version: "2.0.0",
}

// instructions is surfaced to MCP clients (Claude Code et al.) on connect.
// Tells the LLM what Mneme is for and which tools to reach for first.
// Kept terse on purpose — every token here lives in the client's context.
const instructions = `Mneme is the source of truth for plans, specs, and evolving project knowledge between sessions. Repo-tracked files (CLAUDE.md, ADRs, docs/specs/*.md) live on disk, not in Mneme.

Session start: call get_context_bundle(project) — memory, the active plan, recent decisions and journal in one digest. Session end: append_journal (summary + accomplished + deferred). Record decisions with log_decision as you make them. Store durable facts with set_memory; an empty value deletes the key.

Finding things: search(q, types?) is one ranked query across every content type; websearch syntax (phrases, OR, -exclusion). list_documents filters by project/type/status. get_document only when you need a body. resolve_reference turns a mneme:// reference or bare public id (doc_…) into its entity plus the ids the surgical tools need — never guess what a reference points to.

Writing: push_document creates or fully rewrites by meta.id; pass project_create:true if meta.project does not exist yet. For edits prefer the surgical tools (tick_task, update_task, add_task, update_section, add_section, remove_section, remove_task, advance_phase) — they address blocks by stable id and cost ~100x less than re-pushing a document.

Meta keys: id (required stable slug), title (required), type (plan | report | spec | adr | brainstorm | journal), project, status (todo | in-progress | complete | blocked | archived; default todo), tags[], phase_current + phase_total (integers, plan progress), category, ticket, repo.

Body is {sections:[blocks]}; every block has a type, ids are minted when omitted (returned in 'created'), unknown fields are rejected. Shapes: section {title, content?, children?[]}; text {content}; task-list {title?, tasks:[{id?, title, content?, done}]}; subphase {num, title, session?, description?, tasks[], children?[]} — phases are a flat top-level sequence, never nested in another subphase, nums unique and increasing; callout {variant: info|warn|success|danger|note, title?, content}; code {lang, filename?, content}; diagram {content: mermaid}; table {title?, cols[], rows[][]}; key-value {title?, data{}}.

Prose rules: content/description fields hold inline-markdown paragraphs separated by blank lines; titles are single-line. Markdown lists, # headings, and code fences are rejected inside prose — use a child block (task-list, section, code, table, key-value). code and diagram content keep newlines verbatim.

Plan example: {"meta":{"id":"plan-x","title":"Plan: X","type":"plan","project":"p","phase_current":1,"phase_total":2},"body":{"sections":[{"type":"section","title":"Goal","content":"One sentence."},{"type":"subphase","num":1,"title":"Build","tasks":[{"title":"Write the failing test","done":false}]},{"type":"subphase","num":2,"title":"Ship","tasks":[{"title":"Verify and commit","done":false}]}]}}

Relations: link/unlink manage typed edges (relates-to, implements, supersedes, depends-on, blocks); get_related returns links, mentions, and backlinks. Inline [[mentions]] in document prose register automatically on writes.`

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
