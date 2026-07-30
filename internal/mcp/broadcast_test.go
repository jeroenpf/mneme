package mcp_test

import (
	"sync"
	"testing"

	"github.com/jeroenpf/mneme/internal/live"
)

// recordingBroadcaster captures the live events emitted by tool calls so a
// test can assert the write path broadcasts the right type/id/project.
type recordingBroadcaster struct {
	mu     sync.Mutex
	events []live.Event
}

func (r *recordingBroadcaster) Broadcast(ev live.Event) {
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
}

func (r *recordingBroadcaster) snapshot() []live.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]live.Event, len(r.events))
	copy(out, r.events)
	return out
}

// has reports whether any recorded event matches want on every non-empty
// field of want (empty fields are wildcards).
func (r *recordingBroadcaster) has(want live.Event) bool {
	for _, ev := range r.snapshot() {
		if want.Type != "" && ev.Type != want.Type {
			continue
		}
		if want.ID != "" && ev.ID != want.ID {
			continue
		}
		if want.Project != "" && ev.Project != want.Project {
			continue
		}
		if want.Op != "" && ev.Op != want.Op {
			continue
		}
		return true
	}
	return false
}

// TestWritesBroadcastEvents drives each broadcast choke point end-to-end
// through the MCP client: enqueue-routed (log_decision), and the two
// direct broadcasts (set_memory, archive_document).
func TestWritesBroadcastEvents(t *testing.T) {
	bc := &recordingBroadcaster{}
	cs := newClientWithBroadcaster(t, bc)
	seedProject(t, "apollo")

	// log_decision → decisions event (routed through enqueue).
	var dec struct {
		ID string `json:"id"`
	}
	call(t, cs, "log_decision", map[string]any{
		"title": "Use SSE", "decision": "SSE over WebSockets", "project": "apollo",
	}, &dec)
	if !bc.has(live.Event{Type: "decisions", ID: dec.ID}) {
		t.Errorf("no decisions event for id %q; events=%+v", dec.ID, bc.snapshot())
	}

	// set_memory → memory event carrying the project.
	call(t, cs, "set_memory", map[string]any{
		"scope": "project", "project": "apollo", "key": "stack", "value": "go",
	}, nil)
	if !bc.has(live.Event{Type: "memory", ID: "stack", Project: "apollo"}) {
		t.Errorf("no memory event; events=%+v", bc.snapshot())
	}

	// archive_document → documents event tagged with the op.
	call(t, cs, "push_document", samplePlan("live-doc", "apollo"), nil)
	call(t, cs, "archive_document", map[string]any{"id": "live-doc"}, nil)
	if !bc.has(live.Event{Type: "documents", ID: "live-doc", Op: "archive_document"}) {
		t.Errorf("no archive event; events=%+v", bc.snapshot())
	}
}

// TestDocumentEditsBroadcastEditedBlock drives P3: every block-editing doc
// tool broadcasts a documents event naming the changed block (BlockID) and
// the tool (Op), so the viewer can flash exactly that block. advance_phase
// and update_document_meta touch no single block, so their BlockID is empty.
func TestDocumentEditsBroadcastEditedBlock(t *testing.T) {
	bc := &recordingBroadcaster{}
	cs := newClientWithBroadcaster(t, bc)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("d1", "apollo"), nil)

	wantBlock := func(op, blockID string) {
		t.Helper()
		want := live.Event{Type: "documents", ID: "d1", BlockID: blockID, Op: op}
		if !bc.has(want) {
			t.Errorf("%s: no event %+v; events=%+v", op, want, bc.snapshot())
		}
	}

	call(t, cs, "tick_task", map[string]any{"doc_id": "d1", "task_id": "t-001"}, nil)
	wantBlock("tick_task", "t-001")

	call(t, cs, "update_task", map[string]any{
		"doc_id": "d1", "task_id": "t-002", "patch": map[string]any{"title": "Renamed"},
	}, nil)
	wantBlock("update_task", "t-002")

	call(t, cs, "add_task", map[string]any{
		"doc_id": "d1", "section_id": "sp-1-1",
		"task": map[string]any{"id": "t-003", "title": "New task", "done": false},
	}, nil)
	wantBlock("add_task", "t-003")

	call(t, cs, "remove_task", map[string]any{"doc_id": "d1", "task_id": "t-003"}, nil)
	wantBlock("remove_task", "t-003")

	call(t, cs, "update_section", map[string]any{
		"doc_id": "d1", "section_id": "overview", "patch": map[string]any{"title": "Overview v2"},
	}, nil)
	wantBlock("update_section", "overview")

	call(t, cs, "add_section", map[string]any{
		"doc_id": "d1", "section": map[string]any{"id": "sec-2", "type": "section", "title": "Extra"},
	}, nil)
	wantBlock("add_section", "sec-2")

	call(t, cs, "remove_section", map[string]any{"doc_id": "d1", "section_id": "sec-2"}, nil)
	wantBlock("remove_section", "sec-2")

	call(t, cs, "advance_phase", map[string]any{"doc_id": "d1"}, nil)
	wantBlock("advance_phase", "")

	call(t, cs, "update_document_meta", map[string]any{
		"id": "d1", "meta": samplePlan("d1", "apollo")["meta"],
	}, nil)
	wantBlock("update_document_meta", "")
}
