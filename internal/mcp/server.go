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

push_document is upsert-by-meta.id — reserve it for new documents or full rewrites.

Repo-tracked files (CLAUDE.md, ADRs, READMEs, .architecture/specs/*.md) are not in Mneme; read those from disk.`

// Server holds the SDK Server plus the dependencies its tool handlers
// close over. It's safe to share across requests — the SDK manages
// per-session state inside its own machinery.
type Server struct {
	sdk   *sdk.Server
	tools *tools
}

// New constructs a Server with every Phase 1.4 tool registered.
func New(st store.Store) *Server {
	s := &Server{
		sdk:   sdk.NewServer(implementation, &sdk.ServerOptions{Instructions: instructions}),
		tools: &tools{store: st},
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
// (tools_*.go) can stay focused on input/output shapes.
type tools struct {
	store store.Store
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
