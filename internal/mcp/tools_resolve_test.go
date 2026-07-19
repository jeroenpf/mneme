package mcp_test

import (
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/ids"
	"github.com/jeroenpfeil/mneme/internal/models"
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
