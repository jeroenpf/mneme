package mcp_test

import (
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
)

// resolveOut mirrors resolve_reference's structured output for decoding.
type resolveOut struct {
	Kind      string          `json:"kind"`
	Reference string          `json:"reference"`
	TargetID  string          `json:"target_id"`
	Document  *docSummaryTest `json:"document"`
	Content   map[string]any  `json:"content"`
}

// docSummaryTest is the owning-document summary carried by a block/task hit.
type docSummaryTest struct {
	ID       string `json:"id"`
	PublicID string `json:"public_id"`
	Title    string `json:"title"`
}

// samplePlanIDless is a document whose blocks and tasks omit ids, so push mints
// blk_/task_ public ids the reference resolver can then address.
func samplePlanIDless(id, project string) map[string]any {
	return map[string]any{
		"meta": map[string]any{"id": id, "title": "Ref Fixture", "type": "spec", "project": project},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "section", "title": "Intro", "children": []any{
				map[string]any{"type": "text", "content": "hello"},
			}},
			map[string]any{"type": "task-list", "title": "Steps", "tasks": []any{
				map[string]any{"title": "do a thing"},
			}},
		}},
	}
}

func TestResolveReferenceDocument(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlanIDless("refdoc", "apollo"), &pushed)
	if !ids.ValidFor(ids.KindDocument, pushed.PublicID) {
		t.Fatalf("push did not return a doc_ public id: %q", pushed.PublicID)
	}

	var out resolveOut
	call(t, cs, "resolve_reference", map[string]any{"ref": "mneme://document/" + pushed.PublicID}, &out)
	if out.Kind != "document" {
		t.Errorf("kind = %q, want document", out.Kind)
	}
	if out.TargetID != pushed.PublicID {
		t.Errorf("target_id = %q, want %q", out.TargetID, pushed.PublicID)
	}
	// Content is the full document — its slug and body sections come through.
	if got, _ := out.Content["id"].(string); got != "refdoc" {
		t.Errorf("content.id = %q, want refdoc", got)
	}
	if body, _ := out.Content["body"].(map[string]any); body == nil || body["sections"] == nil {
		t.Errorf("content should carry the document body: %+v", out.Content)
	}
}

func TestResolveReferenceBareID(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlanIDless("baredoc", "apollo"), &pushed)

	// A bare public id (no mneme:// scheme) resolves just like the full URI.
	var out resolveOut
	call(t, cs, "resolve_reference", map[string]any{"ref": pushed.PublicID}, &out)
	if out.Kind != "document" || out.TargetID != pushed.PublicID {
		t.Errorf("bare-id resolve = %+v", out)
	}
}

func TestResolveReferenceBlockAndTask(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var pushed struct {
		PublicID string      `json:"public_id"`
		Created  []createdID `json:"created"`
	}
	call(t, cs, "push_document", samplePlanIDless("nested", "apollo"), &pushed)

	var blockID, taskID string
	for _, c := range pushed.Created {
		if c.Kind == "block" && blockID == "" {
			blockID = c.ID
		}
		if c.Kind == "task" && taskID == "" {
			taskID = c.ID
		}
	}
	if blockID == "" || taskID == "" {
		t.Fatalf("expected minted block and task ids, got %+v", pushed.Created)
	}

	// Block reference resolves to the block node, carrying its owning document.
	var block resolveOut
	call(t, cs, "resolve_reference",
		map[string]any{"ref": "mneme://document/" + pushed.PublicID + "/block/" + blockID}, &block)
	if block.Kind != "block" || block.TargetID != blockID {
		t.Errorf("block resolve = %+v", block)
	}
	if block.Document == nil || block.Document.ID != "nested" || block.Document.PublicID != pushed.PublicID {
		t.Errorf("block owning document = %+v", block.Document)
	}
	if got, _ := block.Content["id"].(string); got != blockID {
		t.Errorf("block content id = %q, want %q", got, blockID)
	}

	// Task reference resolves to the task node.
	var task resolveOut
	call(t, cs, "resolve_reference",
		map[string]any{"ref": "mneme://document/" + pushed.PublicID + "/task/" + taskID}, &task)
	if task.Kind != "task" || task.TargetID != taskID {
		t.Errorf("task resolve = %+v", task)
	}
	if task.Document == nil || task.Document.ID != "nested" {
		t.Errorf("task owning document = %+v", task.Document)
	}
}

func TestResolveReferenceSemanticNested(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	// samplePlan carries legacy semantic block/task ids (overview, t-001), not
	// generated blk_/task_ ids — the pre-migration reality P6 must still resolve.
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), &pushed)

	var block resolveOut
	call(t, cs, "resolve_reference",
		map[string]any{"ref": "mneme://document/" + pushed.PublicID + "/block/overview"}, &block)
	if block.Kind != "block" || block.TargetID != "overview" {
		t.Errorf("semantic block resolve = %+v", block)
	}
	if block.Document == nil || block.Document.ID != "vehicle-api" {
		t.Errorf("semantic block owning doc = %+v", block.Document)
	}
	if block.Reference != "mneme://document/"+pushed.PublicID+"/block/overview" {
		t.Errorf("semantic block reference = %q", block.Reference)
	}

	var task resolveOut
	call(t, cs, "resolve_reference",
		map[string]any{"ref": "mneme://document/" + pushed.PublicID + "/task/t-001"}, &task)
	if task.Kind != "task" || task.TargetID != "t-001" {
		t.Errorf("semantic task resolve = %+v", task)
	}
	if got, _ := task.Content["title"].(string); got != "Init Go module" {
		t.Errorf("semantic task content title = %q", got)
	}
}

func TestDocToolsAcceptPublicID(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), &pushed)

	// get_document accepts the doc_ public id, not only the slug.
	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": pushed.PublicID}, &doc)
	if doc.ID != "vehicle-api" {
		t.Errorf("get_document by public id = %q, want vehicle-api", doc.ID)
	}

	// A surgical edit accepts the public id as doc_id too — so an agent can act
	// on a resolved reference's document id directly.
	var ticked struct {
		Done bool `json:"done"`
	}
	call(t, cs, "tick_task", map[string]any{"doc_id": pushed.PublicID, "task_id": "t-001"}, &ticked)
	if !ticked.Done {
		t.Errorf("tick_task via public id: done=%v, want true", ticked.Done)
	}
}

func TestResolveReferenceDecision(t *testing.T) {
	cs := newClient(t)
	var dec models.Decision
	call(t, cs, "log_decision", map[string]any{"title": "Adopt RRF", "decision": "fuse via reciprocal rank"}, &dec)
	if !ids.ValidFor(ids.KindDecision, dec.PublicID) {
		t.Fatalf("log_decision public id = %q", dec.PublicID)
	}

	var out resolveOut
	call(t, cs, "resolve_reference", map[string]any{"ref": "mneme://decision/" + dec.PublicID}, &out)
	if out.Kind != "decision" || out.TargetID != dec.PublicID {
		t.Errorf("decision resolve = %+v", out)
	}
	if got, _ := out.Content["title"].(string); got != "Adopt RRF" {
		t.Errorf("decision content title = %q", got)
	}
}

// TestReferenceRoundTripToSurgicalUpdate walks the whole path the reference
// feature promises: a reference a user copies from the UI, resolved via
// resolve_reference, whose returned ids drive a surgical tick_task — the edit
// then visible on re-read.
func TestReferenceRoundTripToSurgicalUpdate(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlan("vehicle-api", "apollo"), &pushed)

	// The reference RefChip would copy for task t-001.
	ref := "mneme://document/" + pushed.PublicID + "/task/t-001"

	var resolved resolveOut
	call(t, cs, "resolve_reference", map[string]any{"ref": ref}, &resolved)
	if resolved.Kind != "task" || resolved.TargetID != "t-001" || resolved.Document == nil {
		t.Fatalf("resolve = %+v", resolved)
	}

	// Surgical update using only what resolve_reference returned.
	var ticked struct {
		Done bool `json:"done"`
	}
	call(t, cs, "tick_task",
		map[string]any{"doc_id": resolved.Document.ID, "task_id": resolved.TargetID}, &ticked)
	if !ticked.Done {
		t.Fatalf("tick via resolved ids: done=%v, want true", ticked.Done)
	}

	// Visible on re-read.
	var doc models.Document
	call(t, cs, "get_document", map[string]any{"id": resolved.Document.ID}, &doc)
	sections := doc.Body["sections"].([]any)
	subphase := sections[1].(map[string]any)
	task := subphase["tasks"].([]any)[0].(map[string]any)
	if task["id"] != "t-001" || task["done"] != true {
		t.Errorf("t-001 done not persisted: %+v", task)
	}
}

func TestResolveReferenceErrors(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	var pushed struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "push_document", samplePlanIDless("errdoc", "apollo"), &pushed)

	cases := map[string]struct {
		ref  string
		want string // substring the error should mention
	}{
		"empty":           {"", "required"},
		"garbage":         {"not a reference", "reference"},
		"unknown kind":    {"mneme://banana/doc_000000000000", "kind"},
		"absent document": {"mneme://document/doc_000000000000", "no document"},
		"absent decision": {"mneme://decision/dec_000000000000", "no decision"},
		"missing block":   {"mneme://document/" + pushed.PublicID + "/block/blk_000000000000", "no block"},
		"bare nested id":  {"blk_000000000000", "document"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			msg := callExpectError(t, cs, "resolve_reference", map[string]any{"ref": tc.ref})
			if !strings.Contains(strings.ToLower(msg), tc.want) {
				t.Errorf("error = %q, want it to mention %q", msg, tc.want)
			}
		})
	}
}
