package ids_test

import (
	"testing"

	"github.com/jeroenpfeil/mneme/internal/ids"
)

func TestRefFormatsTopLevelEntities(t *testing.T) {
	cases := []struct {
		kind ids.Kind
		id   string
		want string
	}{
		{ids.KindProject, "prj_000000000000", "mneme://project/prj_000000000000"},
		{ids.KindDocument, "doc_000000000000", "mneme://document/doc_000000000000"},
		{ids.KindDecision, "dec_000000000000", "mneme://decision/dec_000000000000"},
		{ids.KindJournal, "jrnl_00000000000a", "mneme://journal/jrnl_00000000000a"},
		{ids.KindSnippet, "snip_00000000000a", "mneme://snippet/snip_00000000000a"},
		{ids.KindSolution, "sol_00000000000a", "mneme://solution/sol_00000000000a"},
	}
	for _, tc := range cases {
		got, err := ids.Ref(tc.kind, tc.id)
		if err != nil {
			t.Errorf("Ref(%s, %q): %v", tc.kind, tc.id, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Ref(%s, %q) = %q, want %q", tc.kind, tc.id, got, tc.want)
		}
	}
}

func TestRefRejectsPrefixEntityMismatch(t *testing.T) {
	if _, err := ids.Ref(ids.KindDocument, "prj_000000000000"); err == nil {
		t.Error("Ref(document, prj_…) should reject the entity/prefix mismatch")
	}
	if _, err := ids.Ref(ids.KindProject, "not-an-id"); err == nil {
		t.Error("Ref(project, malformed) should error")
	}
}

func TestRefRejectsNestedKinds(t *testing.T) {
	if _, err := ids.Ref(ids.KindBlock, "blk_000000000000"); err == nil {
		t.Error("Ref(block, …) should direct callers to RefBlock")
	}
	if _, err := ids.Ref(ids.KindTask, "task_000000000000"); err == nil {
		t.Error("Ref(task, …) should direct callers to RefTask")
	}
}

func TestRefBlockAndRefTaskNestUnderTheDocument(t *testing.T) {
	block, err := ids.RefBlock("doc_000000000000", "blk_111111111111")
	if err != nil {
		t.Fatalf("RefBlock: %v", err)
	}
	if want := "mneme://document/doc_000000000000/block/blk_111111111111"; block != want {
		t.Errorf("RefBlock = %q, want %q", block, want)
	}

	task, err := ids.RefTask("doc_000000000000", "task_111111111111")
	if err != nil {
		t.Fatalf("RefTask: %v", err)
	}
	if want := "mneme://document/doc_000000000000/task/task_111111111111"; task != want {
		t.Errorf("RefTask = %q, want %q", task, want)
	}
}

func TestRefBlockRejectsMismatchedIDs(t *testing.T) {
	// Owner must be a document.
	if _, err := ids.RefBlock("prj_000000000000", "blk_111111111111"); err == nil {
		t.Error("RefBlock with a non-document owner should error")
	}
	// The nested id must actually be a block.
	if _, err := ids.RefBlock("doc_000000000000", "task_111111111111"); err == nil {
		t.Error("RefBlock with a task id should error")
	}
	// And the task variant must reject a block id.
	if _, err := ids.RefTask("doc_000000000000", "blk_111111111111"); err == nil {
		t.Error("RefTask with a block id should error")
	}
}
