package mcp_test

import (
	"sync"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/live"
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
// through the MCP client: enqueue-routed (log_decision), and the three
// direct broadcasts (set_memory, set_env, archive_document).
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

	// set_env → env event carrying the project (env view is project-scoped).
	call(t, cs, "set_env", map[string]any{
		"project": "apollo", "key": "API_PORT", "value": "8443",
	}, nil)
	if !bc.has(live.Event{Type: "env", ID: "API_PORT", Project: "apollo"}) {
		t.Errorf("no env event; events=%+v", bc.snapshot())
	}

	// archive_document → documents event tagged with the op.
	call(t, cs, "push_document", samplePlan("live-doc", "apollo"), nil)
	call(t, cs, "archive_document", map[string]any{"id": "live-doc"}, nil)
	if !bc.has(live.Event{Type: "documents", ID: "live-doc", Op: "archive_document"}) {
		t.Errorf("no archive event; events=%+v", bc.snapshot())
	}
}
