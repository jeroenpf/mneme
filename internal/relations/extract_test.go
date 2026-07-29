package relations

import (
	"reflect"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
)

// ExtractRefs walks every string in the body — prose, task fields, table
// cells, code — collects doc_/dec_/snip_/sol_/jrnl_ public ids and [[slug]]
// wikilinks, drops self-references, and returns a sorted deduped list.
// blk_/task_/proj_ ids are not relation targets.
func TestExtractRefs(t *testing.T) {
	doc := &models.Document{
		ID:       "plan-self",
		PublicID: "doc_selfselfself",
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "s", "title": "S",
				"content": "See doc_abc123xyz and our own doc_selfselfself plus [[plan-self]].",
				"children": []any{
					map[string]any{"type": "subphase", "id": "sp", "num": "1", "title": "P", "tasks": []any{
						map[string]any{"id": "t1", "title": "task", "content": "graduates [[plan-foo]] when done, see snip_a1b2c3d4 twice: snip_a1b2c3d4"},
					}},
					map[string]any{"type": "table", "id": "tb", "cols": []any{"a"}, "rows": []any{[]any{"decision dec_q1w2e3r4"}}},
					map[string]any{"type": "code", "id": "c", "lang": "go", "content": "// sol_z9y8x7w6 and jrnl_p0o9i8u7\nblk_ignoreme0 task_ignoreme0 proj_ignoreme0"},
				}},
		}},
	}
	got := ExtractRefs(doc)
	want := []string{"dec_q1w2e3r4", "doc_abc123xyz", "jrnl_p0o9i8u7", "plan-foo", "snip_a1b2c3d4", "sol_z9y8x7w6"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractRefs = %v, want %v", got, want)
	}
}

func TestMentionRows(t *testing.T) {
	doc := &models.Document{
		ID: "plan-self", PublicID: "doc_selfselfself",
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "doc_abc123xyz and [[future-doc]]"},
		}},
	}
	rows := MentionRows(doc)
	if len(rows) != 2 {
		t.Fatalf("MentionRows = %d rows, want 2: %+v", len(rows), rows)
	}
	if rows[0].ToRef != "doc_abc123xyz" || rows[0].ToID == nil || *rows[0].ToID != "doc_abc123xyz" {
		t.Errorf("public-id row must resolve ToID: %+v", rows[0])
	}
	if rows[1].ToRef != "future-doc" || rows[1].ToID != nil {
		t.Errorf("slug row must stay dangling: %+v", rows[1])
	}
	for _, r := range rows {
		if r.FromID != "doc_selfselfself" || r.RelType != "mentions" || r.Origin != "auto" {
			t.Errorf("row metadata: %+v", r)
		}
	}
}
