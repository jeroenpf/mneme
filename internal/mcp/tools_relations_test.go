package mcp_test

import (
	"strings"
	"testing"
)

// The relations tools: auto mentions appear from document writes (the
// command-path scanner), link/unlink manage explicit typed edges, and
// get_related returns the enriched bundle with backlinks.
func TestRelationsTools(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "proj")

	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "plan-b", "title": "Plan B", "type": "plan", "project": "proj"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "standalone"},
		}},
	}, nil)
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "plan-a", "title": "Plan A", "type": "plan", "project": "proj"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "builds on [[plan-b]]"},
		}},
	}, &pushed)

	type entry struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Title     string `json:"title"`
		RelType   string `json:"rel_type"`
		Direction string `json:"direction"`
	}
	var bundle struct {
		Links       []entry `json:"links"`
		Mentions    []entry `json:"mentions"`
		MentionedBy []entry `json:"mentioned_by"`
	}

	// The push of plan-a synced its mention of plan-b — a backlink for B.
	call(t, cs, "get_related", map[string]any{"ref": "plan-b"}, &bundle)
	if len(bundle.MentionedBy) != 1 || bundle.MentionedBy[0].Title != "Plan A" {
		t.Fatalf("plan-b mentioned_by: %+v", bundle.MentionedBy)
	}

	// Explicit typed link, visible from both ends with correct directions.
	call(t, cs, "link", map[string]any{"from": "plan-a", "to": "plan-b", "rel_type": "implements"}, nil)
	call(t, cs, "get_related", map[string]any{"ref": "plan-a"}, &bundle)
	if len(bundle.Links) != 1 || bundle.Links[0].Direction != "out" || bundle.Links[0].RelType != "implements" {
		t.Fatalf("plan-a links: %+v", bundle.Links)
	}
	call(t, cs, "get_related", map[string]any{"ref": "plan-b"}, &bundle)
	if len(bundle.Links) != 1 || bundle.Links[0].Direction != "in" || bundle.Links[0].Title != "Plan A" {
		t.Fatalf("plan-b links: %+v", bundle.Links)
	}

	// mentions is scanner-owned.
	msg := callExpectError(t, cs, "link", map[string]any{"from": "plan-a", "to": "plan-b", "rel_type": "mentions"})
	if !strings.Contains(msg, "rel_type") {
		t.Errorf("link(mentions) error should name rel_type, got %q", msg)
	}

	var removed struct {
		Removed int64 `json:"removed"`
	}
	call(t, cs, "unlink", map[string]any{"from": "plan-a", "to": "plan-b"}, &removed)
	if removed.Removed != 1 {
		t.Fatalf("unlink removed %d, want 1", removed.Removed)
	}
	call(t, cs, "get_related", map[string]any{"ref": "plan-a"}, &bundle)
	if len(bundle.Links) != 0 || len(bundle.Mentions) != 1 {
		t.Fatalf("after unlink: %+v", bundle)
	}
}
