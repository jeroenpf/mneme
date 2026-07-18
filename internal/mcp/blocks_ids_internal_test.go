package mcp

import (
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/ids"
)

// body builds a {sections:[...]} body from top-level blocks.
func body(blocks ...any) map[string]any {
	return map[string]any{"sections": blocks}
}

func TestNormalizeBodyIDsAssignsMissingBlockAndTaskIDs(t *testing.T) {
	b := body(
		map[string]any{"type": "section", "id": "keep", "children": []any{
			map[string]any{"type": "text", "content": "no id here"},
		}},
		map[string]any{"type": "task-list", "tasks": []any{
			map[string]any{"title": "task without id"},
		}},
	)

	created, err := normalizeBodyIDs(b)
	if err != nil {
		t.Fatalf("normalizeBodyIDs: %v", err)
	}
	// text block, task-list block, and the task all lacked ids.
	if len(created) != 3 {
		t.Fatalf("created = %d ids, want 3: %+v", len(created), created)
	}

	sections := b["sections"].([]any)
	sec := sections[0].(map[string]any)
	if sec["id"] != "keep" {
		t.Errorf("supplied section id changed to %v", sec["id"])
	}
	text := sec["children"].([]any)[0].(map[string]any)
	if id, _ := text["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("text block id = %q, want a valid blk_ id", id)
	}
	tl := sections[1].(map[string]any)
	if id, _ := tl["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("task-list id = %q, want a valid blk_ id", id)
	}
	task := tl["tasks"].([]any)[0].(map[string]any)
	if id, _ := task["id"].(string); !ids.ValidFor(ids.KindTask, id) {
		t.Errorf("task id = %q, want a valid task_ id", id)
	}
}

func TestNormalizeBodyIDsPreservesSuppliedIDs(t *testing.T) {
	b := body(
		map[string]any{"type": "text", "id": "alpha", "content": "a"},
		map[string]any{"type": "text", "id": "beta", "content": "b"},
	)
	created, err := normalizeBodyIDs(b)
	if err != nil {
		t.Fatalf("normalizeBodyIDs: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("created = %+v, want none (all ids supplied)", created)
	}
}

func TestNormalizeBodyIDsRejectsDuplicateBlockIDs(t *testing.T) {
	b := body(
		map[string]any{"type": "text", "id": "dup", "content": "a"},
		map[string]any{"type": "text", "id": "dup", "content": "b"},
	)
	err := normalizeBodyIDs2Err(t, b)
	if !strings.Contains(err.Error(), "unique") || !strings.Contains(err.Error(), "dup") {
		t.Errorf("error = %q, want it to name the duplicate id and the uniqueness rule", err)
	}
}

func TestNormalizeBodyIDsRejectsBlockTaskNamespaceCollision(t *testing.T) {
	// A block and a task sharing an id break addressing just as two blocks would.
	b := body(
		map[string]any{"type": "text", "id": "x", "content": "a block"},
		map[string]any{"type": "task-list", "id": "tl", "tasks": []any{
			map[string]any{"id": "x", "title": "a task"},
		}},
	)
	if err := normalizeBodyIDs2Err(t, b); err == nil {
		t.Fatal("a block id colliding with a task id must be rejected")
	}
}

func TestNormalizeBodyIDsRejectsNonStringID(t *testing.T) {
	b := body(
		map[string]any{"type": "text", "id": 123, "content": "a"},
	)
	if err := normalizeBodyIDs2Err(t, b); err == nil {
		t.Fatal("a non-string id must be rejected")
	}
}

func TestNormalizeBodyIDsTreatsEmptyStringAsMissing(t *testing.T) {
	b := body(map[string]any{"type": "text", "id": "", "content": "a"})
	created, err := normalizeBodyIDs(b)
	if err != nil {
		t.Fatalf("normalizeBodyIDs: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want the empty-id block to be assigned one", created)
	}
}

func TestResolveIDsRejectsCollisionWithExistingDoc(t *testing.T) {
	taken := map[string]bool{"taken-id": true}
	nodes := []idNode{{node: map[string]any{"type": "text", "id": "taken-id"}, kind: ids.KindBlock, path: "block"}}
	if _, err := resolveIDs(nodes, taken); err == nil {
		t.Fatal("a supplied id already used in the document must be rejected")
	}
}

func TestResolveIDsMintsAroundTakenSet(t *testing.T) {
	taken := map[string]bool{"other": true}
	node := map[string]any{"type": "text"}
	created, err := resolveIDs([]idNode{{node: node, kind: ids.KindBlock, path: "block"}}, taken)
	if err != nil {
		t.Fatalf("resolveIDs: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want one minted id", created)
	}
	if id, _ := node["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("minted id = %q, want a valid blk_ id", id)
	}
}

// normalizeBodyIDs2Err runs normalizeBodyIDs and fails the test unless it
// returns an error, returning that error for message assertions.
func normalizeBodyIDs2Err(t *testing.T, b map[string]any) error {
	t.Helper()
	_, err := normalizeBodyIDs(b)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	return err
}
