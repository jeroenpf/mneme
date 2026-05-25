package mcp_test

import (
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// samplePlan returns a meta+body pair shaped like the Mneme plan format
// — one subphase with two tasks plus a nested section. Used as a
// fixture across most editing tests.
func samplePlan(id, project string) map[string]any {
	return map[string]any{
		"meta": map[string]any{
			"id":      id,
			"title":   "Vehicle Listing API",
			"type":    "plan",
			"project": project,
			"phases": []any{
				map[string]any{"title": "Foundation", "status": "done"},
				map[string]any{"title": "API Layer", "status": "wip"},
				map[string]any{"title": "Frontend", "status": "todo"},
			},
		},
		"body": map[string]any{
			"sections": []any{
				map[string]any{
					"type": "section", "id": "overview", "title": "Overview",
					"children": []any{
						map[string]any{"type": "text", "id": "p1", "content": "Adds paginated listings."},
					},
				},
				map[string]any{
					"type": "subphase", "id": "sp-1-1", "title": "Scaffolding",
					"tasks": []any{
						map[string]any{"id": "t-001", "title": "Init Go module", "done": false},
						map[string]any{"id": "t-002", "title": "Docker Compose", "done": false},
					},
				},
			},
		},
	}
}

func TestPushDocumentCreatesThenUpserts(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var first models.Document
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), &first)
	if first.ID != "vehicle-api" {
		t.Errorf("created id: got %q, want vehicle-api", first.ID)
	}
	if first.Title != "Vehicle Listing API" {
		t.Errorf("title: got %q", first.Title)
	}

	// Re-push with a different title — should upsert (no duplicate id error).
	payload := samplePlan("vehicle-api", "apollo")
	payload["meta"].(map[string]any)["title"] = "Vehicle Listing API v2"
	var second models.Document
	call(t, cs, "push_document", payload, &second)
	if second.Title != "Vehicle Listing API v2" {
		t.Errorf("upsert title: got %q", second.Title)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("upsert lost created_at: first=%v second=%v", first.CreatedAt, second.CreatedAt)
	}
}

func TestPushDocumentRejectsInvalidBlockType(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	payload := samplePlan("vehicle-api", "apollo")
	payload["body"].(map[string]any)["sections"] = []any{
		map[string]any{"type": "nonexistent-block", "id": "bad"},
	}
	msg := callExpectError(t, cs, "push_document", payload)
	if msg == "" {
		t.Errorf("expected error mentioning invalid type, got empty message")
	}
}

func TestPushDocumentRequiresMetaID(t *testing.T) {
	cs := newClient(t)
	payload := samplePlan("", "")
	delete(payload["meta"].(map[string]any), "id")
	delete(payload["meta"].(map[string]any), "project")
	msg := callExpectError(t, cs, "push_document", payload)
	if msg == "" {
		t.Errorf("expected meta.id error, got empty")
	}
}

func TestListAndGetDocument(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	seedProject(t, "hermes")

	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)
	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{
			"id": "pricing-engine", "title": "Pricing Engine", "type": "plan", "project": "hermes",
		},
		"body": map[string]any{"sections": []any{}},
	}, nil)

	// list without filters should see both.
	var all struct {
		Items []map[string]any `json:"items"`
	}
	call(t, cs, "list_documents", map[string]any{}, &all)
	if len(all.Items) != 2 {
		t.Errorf("list all: got %d, want 2", len(all.Items))
	}
	for _, it := range all.Items {
		if _, hasBody := it["body"]; hasBody {
			t.Errorf("list_documents leaked body: %+v", it)
		}
	}

	// filter by project
	var filtered struct {
		Items []map[string]any `json:"items"`
	}
	call(t, cs, "list_documents", map[string]any{"project": "apollo"}, &filtered)
	if len(filtered.Items) != 1 || filtered.Items[0]["id"] != "vehicle-api" {
		t.Errorf("filter: got %+v", filtered.Items)
	}

	// get_document returns full body
	var fetched models.Document
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &fetched)
	if fetched.Body == nil {
		t.Errorf("get_document: body missing")
	}
	if _, ok := fetched.Body["sections"]; !ok {
		t.Errorf("get_document: sections missing from body")
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "get_document", map[string]any{"id": "nope"})
	if msg == "" {
		t.Errorf("expected not-found message")
	}
}

func TestSearchDocumentsRankAndOR(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)
	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{
			"id": "pricing-engine", "title": "Pricing Engine", "type": "plan", "project": "apollo",
		},
		"body": map[string]any{"sections": []any{}},
	}, nil)

	var hits struct {
		Items []map[string]any `json:"items"`
	}
	call(t, cs, "search_documents", map[string]any{"q": "vehicle"}, &hits)
	if len(hits.Items) != 1 || hits.Items[0]["id"] != "vehicle-api" {
		t.Errorf("simple search: %+v", hits.Items)
	}

	// websearch OR.
	call(t, cs, "search_documents", map[string]any{"q": "vehicle OR pricing"}, &hits)
	if len(hits.Items) != 2 {
		t.Errorf("OR search: got %d, want 2", len(hits.Items))
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "search_documents", map[string]any{})
	if msg == "" {
		t.Errorf("expected error for empty q")
	}
}

func TestTickAndUpdateTask(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	var first struct {
		Task map[string]any `json:"task"`
	}
	call(t, cs, "tick_task", map[string]any{"doc_id": "vehicle-api", "task_id": "t-001"}, &first)
	if done, _ := first.Task["done"].(bool); !done {
		t.Errorf("tick_task: expected done=true, got %+v", first.Task)
	}

	// Tick again — toggles off.
	var second struct {
		Task map[string]any `json:"task"`
	}
	call(t, cs, "tick_task", map[string]any{"doc_id": "vehicle-api", "task_id": "t-001"}, &second)
	if done, _ := second.Task["done"].(bool); done {
		t.Errorf("tick_task toggle: expected done=false, got %+v", second.Task)
	}

	// update_task patches title and tags.
	var patched struct {
		Task map[string]any `json:"task"`
	}
	call(t, cs, "update_task", map[string]any{
		"doc_id":  "vehicle-api",
		"task_id": "t-002",
		"patch":   map[string]any{"title": "Renamed", "tags": []any{"infra"}},
	}, &patched)
	if patched.Task["title"] != "Renamed" {
		t.Errorf("update_task title: %+v", patched.Task)
	}
	tags, _ := patched.Task["tags"].([]any)
	if len(tags) != 1 || tags[0] != "infra" {
		t.Errorf("update_task tags: %+v", tags)
	}

	// Invalid patch field rejected.
	msg := callExpectError(t, cs, "update_task", map[string]any{
		"doc_id": "vehicle-api", "task_id": "t-002",
		"patch": map[string]any{"banana": "yes"},
	})
	if msg == "" {
		t.Errorf("expected rejection of invalid patch field")
	}
}

func TestAddAndRemoveTask(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	// add_task after t-001 → expect order t-001, t-003, t-002.
	call(t, cs, "add_task", map[string]any{
		"doc_id":        "vehicle-api",
		"section_id":    "sp-1-1",
		"after_task_id": "t-001",
		"task":          map[string]any{"id": "t-003", "title": "Wedge"},
	}, nil)

	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ := doc.Body["sections"].([]any)
	sp := sections[1].(map[string]any)
	tasks, _ := sp["tasks"].([]any)
	ids := []string{}
	for _, raw := range tasks {
		tm := raw.(map[string]any)
		ids = append(ids, tm["id"].(string))
	}
	want := []string{"t-001", "t-003", "t-002"}
	for i, id := range want {
		if i >= len(ids) || ids[i] != id {
			t.Errorf("task order: got %v, want %v", ids, want)
			break
		}
	}

	// remove the middle one.
	call(t, cs, "remove_task", map[string]any{"doc_id": "vehicle-api", "task_id": "t-003"}, nil)
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ = doc.Body["sections"].([]any)
	sp = sections[1].(map[string]any)
	tasks, _ = sp["tasks"].([]any)
	if len(tasks) != 2 {
		t.Errorf("after remove: %d tasks, want 2", len(tasks))
	}
}

func TestSectionUpdateAddRemove(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	// update overview section title.
	call(t, cs, "update_section", map[string]any{
		"doc_id":     "vehicle-api",
		"section_id": "overview",
		"patch":      map[string]any{"title": "Overview Reframed"},
	}, nil)

	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ := doc.Body["sections"].([]any)
	if sections[0].(map[string]any)["title"] != "Overview Reframed" {
		t.Errorf("section title not updated: %+v", sections[0])
	}

	// Cannot patch id.
	msg := callExpectError(t, cs, "update_section", map[string]any{
		"doc_id": "vehicle-api", "section_id": "overview",
		"patch": map[string]any{"id": "renamed"},
	})
	if msg == "" {
		t.Errorf("expected error patching protected field id")
	}

	// add a new top-level section after overview.
	call(t, cs, "add_section", map[string]any{
		"doc_id":           "vehicle-api",
		"after_section_id": "overview",
		"section": map[string]any{
			"type": "callout", "id": "warn-1", "content": "Heads up.",
		},
	}, nil)
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ = doc.Body["sections"].([]any)
	if len(sections) != 3 || sections[1].(map[string]any)["id"] != "warn-1" {
		t.Errorf("add_section order: got %+v", sectionIDs(sections))
	}

	// add_section rejects invalid type.
	msg = callExpectError(t, cs, "add_section", map[string]any{
		"doc_id":  "vehicle-api",
		"section": map[string]any{"type": "what", "id": "x"},
	})
	if msg == "" {
		t.Errorf("expected invalid type rejection")
	}

	// remove the new section (top-level).
	call(t, cs, "remove_section", map[string]any{"doc_id": "vehicle-api", "section_id": "warn-1"}, nil)
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ = doc.Body["sections"].([]any)
	if len(sections) != 2 {
		t.Errorf("after remove top: got %d sections, want 2", len(sections))
	}

	// remove a nested child (p1 inside overview).
	call(t, cs, "remove_section", map[string]any{"doc_id": "vehicle-api", "section_id": "p1"}, nil)
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	sections, _ = doc.Body["sections"].([]any)
	overview := sections[0].(map[string]any)
	children, _ := overview["children"].([]any)
	if len(children) != 0 {
		t.Errorf("nested remove: children=%+v", children)
	}
}

func sectionIDs(sections []any) []string {
	out := []string{}
	for _, raw := range sections {
		b, _ := raw.(map[string]any)
		id, _ := b["id"].(string)
		out = append(out, id)
	}
	return out
}

func TestAdvancePhase(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	// Plan starts: [done, wip, todo] — advance flips API Layer to done, Frontend to wip.
	var out struct {
		CompletedIndex int              `json:"completed_index"`
		NextIndex      *int             `json:"next_index"`
		Doc            *models.Document `json:"doc"`
	}
	call(t, cs, "advance_phase", map[string]any{"doc_id": "vehicle-api"}, &out)
	if out.CompletedIndex != 1 {
		t.Errorf("completed_index: got %d, want 1", out.CompletedIndex)
	}
	if out.NextIndex == nil || *out.NextIndex != 2 {
		t.Errorf("next_index: got %v, want 2", out.NextIndex)
	}
	if out.Doc.PhaseCurrent == nil || *out.Doc.PhaseCurrent != 3 {
		t.Errorf("phase_current: got %v, want 3", out.Doc.PhaseCurrent)
	}

	// Advance again — frontend done, no more todo → status flips to complete.
	// Fresh struct so the omitempty next_index from the previous call
	// doesn't bleed through.
	var final struct {
		CompletedIndex int              `json:"completed_index"`
		NextIndex      *int             `json:"next_index"`
		Doc            *models.Document `json:"doc"`
	}
	call(t, cs, "advance_phase", map[string]any{"doc_id": "vehicle-api"}, &final)
	if final.NextIndex != nil {
		t.Errorf("next_index after final: got %v, want nil", final.NextIndex)
	}
	if final.Doc.Status != models.StatusComplete {
		t.Errorf("status after final advance: %q, want complete", final.Doc.Status)
	}

	// Third advance should error — nothing left.
	msg := callExpectError(t, cs, "advance_phase", map[string]any{"doc_id": "vehicle-api"})
	if msg == "" {
		t.Errorf("expected nothing-to-advance message")
	}
}

func TestArchiveAndUpdateMeta(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), nil)

	call(t, cs, "archive_document", map[string]any{"id": "vehicle-api"}, nil)
	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	if doc.Status != models.StatusArchived {
		t.Errorf("archive: status=%q, want archived", doc.Status)
	}

	// update_document_meta replaces meta wholesale.
	call(t, cs, "update_document_meta", map[string]any{
		"id": "vehicle-api",
		"meta": map[string]any{
			"id":      "vehicle-api",
			"title":   "Renamed",
			"type":    "plan",
			"project": "apollo",
			"tags":    []any{"v2"},
		},
	}, nil)
	call(t, cs, "get_document", map[string]any{"id": "vehicle-api"}, &doc)
	if doc.Title != "Renamed" {
		t.Errorf("renamed title: %q", doc.Title)
	}
	if len(doc.Tags) != 1 || doc.Tags[0] != "v2" {
		t.Errorf("tags: %+v", doc.Tags)
	}
}
