package mcp_test

import (
	"context"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/store"
)

// TestWritePathRecordsRevisions proves the MCP write path appends an
// append-only audit/history snapshot for every document mutation (roadmap P6):
// a push and a surgical edit each leave a revision, newest-first, tagged with
// their operation and affected id.
func TestWritePathRecordsRevisions(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "planx", "title": "Plan X", "type": "plan", "project": "apollo"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp-1", "title": "Phase 1", "tasks": []any{
				map[string]any{"id": "t-1", "title": "do it", "done": false},
			}},
		}},
	}, nil)

	call(t, cs, "tick_task", map[string]any{"doc_id": "planx", "task_id": "t-1"}, nil)

	st := store.NewWithPool(testPool)
	revs, err := st.ListDocumentRevisions(context.Background(), "planx", 0)
	if err != nil {
		t.Fatalf("ListDocumentRevisions: %v", err)
	}
	if len(revs) != 2 {
		t.Fatalf("revision count = %d, want 2 (push + tick)", len(revs))
	}
	// Newest-first: the tick is revision 2, the push revision 1.
	if revs[0].Revision != 2 || revs[0].Op != "tick_task" {
		t.Errorf("latest revision = %+v, want rev 2 op tick_task", revs[0])
	}
	if len(revs[0].TargetIDs) != 1 || revs[0].TargetIDs[0] != "t-1" {
		t.Errorf("tick target ids = %v, want [t-1]", revs[0].TargetIDs)
	}
	if revs[1].Revision != 1 || revs[1].Op != "push_document" {
		t.Errorf("first revision = %+v, want rev 1 op push_document", revs[1])
	}
	if revs[0].Actor != "mcp" {
		t.Errorf("actor = %q, want mcp", revs[0].Actor)
	}
	// The latest snapshot captures the post-write body (task now done).
	if revs[0].Body["sections"] == nil {
		t.Errorf("snapshot body not captured: %+v", revs[0].Body)
	}
}
