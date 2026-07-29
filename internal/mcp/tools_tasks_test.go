package mcp_test

import (
	"strings"
	"testing"
)

// The task write tools enforce the same inline-only rules as push_document:
// block markdown in a task's title/content is rejected with the teaching
// error, while single-line strings with a leading marker stay legal.
func TestTaskToolsRejectBlockMarkdown(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "proj")
	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "d1", "title": "D", "type": "plan", "project": "proj"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp1", "num": "1", "title": "P",
				"tasks": []any{map[string]any{"id": "t1", "title": "ok"}}},
		}},
	}, nil)

	msg := callExpectError(t, cs, "add_task", map[string]any{
		"doc_id": "d1", "section_id": "sp1",
		"task": map[string]any{"title": "bad", "content": "- a\n- b"},
	})
	if !strings.Contains(msg, "inline-only") {
		t.Errorf("add_task error must teach inline-only, got %q", msg)
	}

	msg = callExpectError(t, cs, "update_task", map[string]any{
		"doc_id": "d1", "task_id": "t1",
		"patch": map[string]any{"content": "1. a\n2. b"},
	})
	if !strings.Contains(msg, "inline-only") {
		t.Errorf("update_task error must teach inline-only, got %q", msg)
	}

	// A single-line numbered title renders literally — legal since the
	// detector's single-line exemption.
	call(t, cs, "add_task", map[string]any{
		"doc_id": "d1", "section_id": "sp1",
		"task": map[string]any{"title": "1. config: PublicURL"},
	}, nil)
}
