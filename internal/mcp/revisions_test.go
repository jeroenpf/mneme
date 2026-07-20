package mcp_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// TestDocumentHistoryDiffRestore exercises get_document_history,
// diff_document_revisions, and restore_document_revision end-to-end (roadmap
// P6-t4).
func TestDocumentHistoryDiffRestore(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "planh", "title": "Plan H", "type": "plan", "project": "apollo"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "Overview"},
			map[string]any{"type": "subphase", "id": "sp-1", "title": "Phase 1", "tasks": []any{
				map[string]any{"id": "t-1", "title": "do it", "done": false},
			}},
		}},
	}, nil) // rev 1

	call(t, cs, "tick_task", map[string]any{"doc_id": "planh", "task_id": "t-1"}, nil)                                                               // rev 2
	call(t, cs, "update_section", map[string]any{"doc_id": "planh", "section_id": "overview", "patch": map[string]any{"title": "Overview v2"}}, nil) // rev 3

	// History: newest-first, all three writes attributed.
	var hist struct {
		Revisions []struct {
			Revision int    `json:"revision"`
			Op       string `json:"op"`
		} `json:"revisions"`
	}
	call(t, cs, "get_document_history", map[string]any{"doc_id": "planh"}, &hist)
	if len(hist.Revisions) != 3 {
		t.Fatalf("history len = %d, want 3", len(hist.Revisions))
	}
	if hist.Revisions[0].Op != "update_section" || hist.Revisions[2].Op != "push_document" {
		t.Errorf("history order/ops wrong: %+v", hist.Revisions)
	}

	// Diff rev1 → rev2: only the toggled task changed.
	var diff struct {
		ModifiedIDs []string `json:"modified_ids"`
		AddedIDs    []string `json:"added_ids"`
		RemovedIDs  []string `json:"removed_ids"`
	}
	call(t, cs, "diff_document_revisions", map[string]any{"doc_id": "planh", "from_revision": 1, "to_revision": 2}, &diff)
	if !slices.Equal(diff.ModifiedIDs, []string{"t-1"}) {
		t.Errorf("rev1->rev2 modified = %v, want [t-1]", diff.ModifiedIDs)
	}

	// Diff rev1 → current: both the task and the retitled section changed.
	call(t, cs, "diff_document_revisions", map[string]any{"doc_id": "planh", "from_revision": 1}, &diff)
	if !slices.Contains(diff.ModifiedIDs, "t-1") || !slices.Contains(diff.ModifiedIDs, "overview") {
		t.Errorf("rev1->current modified = %v, want both t-1 and overview", diff.ModifiedIDs)
	}

	// Restore rev1: forward-only new revision (4), content reverts.
	var restored struct {
		RestoredFrom int `json:"restored_from"`
		NewRevision  int `json:"new_revision"`
	}
	call(t, cs, "restore_document_revision", map[string]any{"doc_id": "planh", "revision": 1}, &restored)
	if restored.RestoredFrom != 1 || restored.NewRevision != 4 {
		t.Errorf("restore result = %+v, want restored_from=1 new_revision=4", restored)
	}

	// The restored document matches rev 1: section title back to "Overview".
	var doc struct {
		Body map[string]any `json:"body"`
	}
	call(t, cs, "get_document", map[string]any{"id": "planh"}, &doc)
	sections, _ := doc.Body["sections"].([]any)
	first, _ := sections[0].(map[string]any)
	if first["title"] != "Overview" {
		t.Errorf("restore did not revert section title: %v", first["title"])
	}
}

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

// TestArchiveRecordsRevision proves archive_document routes through the command
// service like every other write: archiving records an append-only
// "archive_document" revision (not a silent status flip), and re-archiving an
// already-archived doc is a safe idempotent no-op that records no further
// revision (closes the roadmap P6 audit gap).
func TestArchiveRecordsRevision(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "push_document", samplePlan("archme", "apollo"), nil) // rev 1

	call(t, cs, "archive_document", map[string]any{"id": "archme"}, nil) // rev 2

	var hist struct {
		Revisions []struct {
			Revision int    `json:"revision"`
			Op       string `json:"op"`
			Actor    string `json:"actor"`
			Status   string `json:"status"`
		} `json:"revisions"`
	}
	call(t, cs, "get_document_history", map[string]any{"doc_id": "archme"}, &hist)
	if len(hist.Revisions) != 2 {
		t.Fatalf("history len = %d, want 2 (push + archive)", len(hist.Revisions))
	}
	if top := hist.Revisions[0]; top.Op != "archive_document" || top.Actor != "mcp" || top.Status != models.StatusArchived {
		t.Errorf("archive revision = %+v, want op=archive_document actor=mcp status=archived", top)
	}

	// Re-archiving an already-archived doc: still ok, but no redundant revision.
	call(t, cs, "archive_document", map[string]any{"id": "archme"}, nil)
	call(t, cs, "get_document_history", map[string]any{"doc_id": "archme"}, &hist)
	if len(hist.Revisions) != 2 {
		t.Fatalf("history len after re-archive = %d, want 2 (idempotent)", len(hist.Revisions))
	}
}
