package mcp_test

import (
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/ids"
	"github.com/jeroenpfeil/mneme/internal/models"
)

// createdID mirrors the tools' created-id outline entry for decoding.
type createdID struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

func TestPushDocumentAssignsMissingBlockAndTaskIDs(t *testing.T) {
	cs := newClient(t)

	payload := map[string]any{
		"meta": map[string]any{"id": "noids", "title": "No IDs", "type": "spec"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "section", "title": "Intro", "children": []any{
				map[string]any{"type": "text", "content": "hello"},
			}},
			map[string]any{"type": "task-list", "title": "Steps", "tasks": []any{
				map[string]any{"title": "first"},
			}},
		}},
	}
	var res struct {
		Created []createdID `json:"created"`
	}
	call(t, cs, "push_document", payload, &res)
	// section, its child text, the task-list, and its task all lacked ids.
	if len(res.Created) != 4 {
		t.Fatalf("created = %d ids, want 4: %+v", len(res.Created), res.Created)
	}

	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "noids"}, &doc)
	sections := doc.Body["sections"].([]any)
	sec := sections[0].(map[string]any)
	if id, _ := sec["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("section id = %q, want a valid blk_ id", id)
	}
	child := sec["children"].([]any)[0].(map[string]any)
	if id, _ := child["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("nested text id = %q, want a valid blk_ id", id)
	}
	tl := sections[1].(map[string]any)
	task := tl["tasks"].([]any)[0].(map[string]any)
	if id, _ := task["id"].(string); !ids.ValidFor(ids.KindTask, id) {
		t.Errorf("task id = %q, want a valid task_ id", id)
	}
}

func TestPushDocumentRejectsDuplicateIDs(t *testing.T) {
	cs := newClient(t)
	payload := map[string]any{
		"meta": map[string]any{"id": "dups", "title": "Dups", "type": "spec"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "same", "content": "a"},
			map[string]any{"type": "text", "id": "same", "content": "b"},
		}},
	}
	msg := callExpectError(t, cs, "push_document", payload)
	if !strings.Contains(msg, "unique") {
		t.Errorf("duplicate-id error = %q, want it to mention the uniqueness rule", msg)
	}
}

func TestAddSectionGeneratesIDWhenOmitted(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	var res struct {
		Section map[string]any `json:"section"`
		Created []createdID    `json:"created"`
	}
	call(t, cs, "add_section", map[string]any{
		"doc_id":  "vehicle-api",
		"section": map[string]any{"type": "text", "content": "appended"},
	}, &res)

	id, _ := res.Section["id"].(string)
	if !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("added section id = %q, want a generated blk_ id", id)
	}
	if len(res.Created) != 1 || res.Created[0].ID != id {
		t.Errorf("created outline = %+v, want the one generated id %q", res.Created, id)
	}
}

func TestAddSectionRejectsCollidingID(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	// "overview" is an existing top-level section id.
	msg := callExpectError(t, cs, "add_section", map[string]any{
		"doc_id":  "vehicle-api",
		"section": map[string]any{"type": "text", "id": "overview", "content": "x"},
	})
	if !strings.Contains(msg, "already used") {
		t.Errorf("collision error = %q, want it to say the id is already used", msg)
	}
}

func TestAddSectionAssignsNestedIDs(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	call(t, cs, "add_section", map[string]any{
		"doc_id": "vehicle-api",
		"section": map[string]any{"type": "section", "title": "Added", "children": []any{
			map[string]any{"type": "text", "content": "nested, no id"},
		}},
	}, nil)

	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections := doc.Body["sections"].([]any)
	added := sections[len(sections)-1].(map[string]any)
	if id, _ := added["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("added section id = %q, want a generated blk_ id", id)
	}
	child := added["children"].([]any)[0].(map[string]any)
	if id, _ := child["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("nested child id = %q, want a generated blk_ id", id)
	}
}

func TestAddTaskGeneratesIDWhenOmitted(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	var res struct {
		Task map[string]any `json:"task"`
	}
	call(t, cs, "add_task", map[string]any{
		"doc_id":     "vehicle-api",
		"section_id": "sp-1-1",
		"task":       map[string]any{"title": "new task, no id"},
	}, &res)
	if id, _ := res.Task["id"].(string); !ids.ValidFor(ids.KindTask, id) {
		t.Errorf("added task id = %q, want a generated task_ id", id)
	}
}

func TestAddTaskRejectsCollidingID(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	// "t-001" is an existing task id in sp-1-1.
	msg := callExpectError(t, cs, "add_task", map[string]any{
		"doc_id":     "vehicle-api",
		"section_id": "sp-1-1",
		"task":       map[string]any{"id": "t-001", "title": "dupe"},
	})
	if !strings.Contains(msg, "already used") {
		t.Errorf("collision error = %q, want it to say the id is already used", msg)
	}
}
