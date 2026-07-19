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

func TestParseRefTopLevelEntities(t *testing.T) {
	cases := []struct {
		ref  string
		want ids.Reference
	}{
		{"mneme://project/prj_000000000000", ids.Reference{Kind: ids.KindProject, ID: "prj_000000000000"}},
		{"mneme://document/doc_000000000000", ids.Reference{Kind: ids.KindDocument, ID: "doc_000000000000"}},
		{"mneme://decision/dec_000000000000", ids.Reference{Kind: ids.KindDecision, ID: "dec_000000000000"}},
		{"mneme://journal/jrnl_00000000000a", ids.Reference{Kind: ids.KindJournal, ID: "jrnl_00000000000a"}},
		{"mneme://snippet/snip_00000000000a", ids.Reference{Kind: ids.KindSnippet, ID: "snip_00000000000a"}},
		{"mneme://solution/sol_00000000000a", ids.Reference{Kind: ids.KindSolution, ID: "sol_00000000000a"}},
	}
	for _, tc := range cases {
		got, err := ids.ParseRef(tc.ref)
		if err != nil {
			t.Errorf("ParseRef(%q): %v", tc.ref, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseRef(%q) = %+v, want %+v", tc.ref, got, tc.want)
		}
	}
}

func TestParseRefNestedBlockAndTask(t *testing.T) {
	block, err := ids.ParseRef("mneme://document/doc_000000000000/block/blk_111111111111")
	if err != nil {
		t.Fatalf("ParseRef(block): %v", err)
	}
	if want := (ids.Reference{Kind: ids.KindBlock, ID: "blk_111111111111", DocID: "doc_000000000000"}); block != want {
		t.Errorf("ParseRef(block) = %+v, want %+v", block, want)
	}

	task, err := ids.ParseRef("mneme://document/doc_000000000000/task/task_111111111111")
	if err != nil {
		t.Fatalf("ParseRef(task): %v", err)
	}
	if want := (ids.Reference{Kind: ids.KindTask, ID: "task_111111111111", DocID: "doc_000000000000"}); task != want {
		t.Errorf("ParseRef(task) = %+v, want %+v", task, want)
	}
}

// ParseRef is the inverse of the Ref/RefBlock/RefTask formatters: anything they
// emit must parse back to the components that produced it.
func TestParseRefRoundTripsWithFormatters(t *testing.T) {
	topLevel := []struct {
		kind ids.Kind
		id   string
	}{
		{ids.KindProject, "prj_000000000000"},
		{ids.KindDocument, "doc_000000000000"},
		{ids.KindDecision, "dec_000000000000"},
		{ids.KindJournal, "jrnl_00000000000a"},
		{ids.KindSnippet, "snip_00000000000a"},
		{ids.KindSolution, "sol_00000000000a"},
	}
	for _, tc := range topLevel {
		ref, err := ids.Ref(tc.kind, tc.id)
		if err != nil {
			t.Fatalf("Ref(%s, %q): %v", tc.kind, tc.id, err)
		}
		got, err := ids.ParseRef(ref)
		if err != nil {
			t.Fatalf("ParseRef(%q): %v", ref, err)
		}
		if want := (ids.Reference{Kind: tc.kind, ID: tc.id}); got != want {
			t.Errorf("round-trip %q = %+v, want %+v", ref, got, want)
		}
	}

	blockRef, _ := ids.RefBlock("doc_000000000000", "blk_111111111111")
	if got, err := ids.ParseRef(blockRef); err != nil || got.Kind != ids.KindBlock || got.ID != "blk_111111111111" || got.DocID != "doc_000000000000" {
		t.Errorf("round-trip block %q = %+v, err=%v", blockRef, got, err)
	}
	taskRef, _ := ids.RefTask("doc_000000000000", "task_111111111111")
	if got, err := ids.ParseRef(taskRef); err != nil || got.Kind != ids.KindTask || got.ID != "task_111111111111" || got.DocID != "doc_000000000000" {
		t.Errorf("round-trip task %q = %+v, err=%v", taskRef, got, err)
	}
}

func TestParseRefRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"empty":                    "",
		"wrong scheme":             "http://document/doc_000000000000",
		"missing scheme":           "document/doc_000000000000",
		"bare id":                  "doc_000000000000",
		"unknown kind segment":     "mneme://banana/doc_000000000000",
		"kind/prefix mismatch":     "mneme://project/doc_000000000000",
		"malformed id":             "mneme://document/doc_bad",
		"missing id":               "mneme://document",
		"trailing slash":           "mneme://document/doc_000000000000/",
		"unexpected extra segment": "mneme://document/doc_000000000000/foo",
		"nested under non-document": "mneme://project/prj_000000000000/block/blk_111111111111",
		"nested wrong child kind":  "mneme://document/doc_000000000000/task/blk_111111111111",
		"nested unknown relation":  "mneme://document/doc_000000000000/comment/blk_111111111111",
		"nested missing child id":  "mneme://document/doc_000000000000/block",
		"nested malformed owner":   "mneme://document/doc_bad/block/blk_111111111111",
	}
	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			if got, err := ids.ParseRef(ref); err == nil {
				t.Errorf("ParseRef(%q) = %+v, want error", ref, got)
			}
		})
	}
}
