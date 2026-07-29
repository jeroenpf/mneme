package mcp_test

import (
	"context"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// lint_documents sweeps ALL docs — archived included — and reports what
// the write path would reject today. The violating doc is inserted at the
// store layer on purpose: it models a legacy doc that predates validation.
func TestLintDocumentsFindsLegacyViolations(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "proj")

	proj := "proj"
	legacy := &models.Document{
		ID: "legacy", Title: "Legacy", Type: "plan", Status: "archived", Project: &proj,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "- a\n- b"},
		}},
	}
	st := store.NewWithPool(testPool)
	if err := st.CreateDocument(context.Background(), legacy); err != nil {
		t.Fatalf("seed legacy doc: %v", err)
	}
	call(t, cs, "push_document", map[string]any{
		"meta": map[string]any{"id": "clean", "title": "Clean", "type": "plan", "project": "proj"},
		"body": map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "ok", "content": "plain prose"},
		}},
	}, nil)

	var out struct {
		Hits []struct {
			BlockID   string `json:"block_id"`
			Path      string `json:"path"`
			Field     string `json:"field"`
			Found     string `json:"found"`
			Excerpt   string `json:"excerpt"`
			DocID     string `json:"doc_id"`
			DocTitle  string `json:"doc_title"`
			DocStatus string `json:"doc_status"`
		} `json:"hits"`
		DocsScanned  int `json:"docs_scanned"`
		DocsWithHits int `json:"docs_with_hits"`
	}
	call(t, cs, "lint_documents", map[string]any{}, &out)

	if out.DocsScanned != 2 || out.DocsWithHits != 1 || len(out.Hits) != 1 {
		t.Fatalf("scanned=%d withHits=%d hits=%d, want 2/1/1", out.DocsScanned, out.DocsWithHits, len(out.Hits))
	}
	h := out.Hits[0]
	if h.DocID != "legacy" || h.DocStatus != "archived" || h.DocTitle != "Legacy" {
		t.Errorf("hit doc fields = %+v", h)
	}
	if h.BlockID != "p" || h.Field != "content" || h.Found != "a list" || h.Path != "body.sections[0]" {
		t.Errorf("hit location = %+v", h)
	}
	if h.Excerpt != "- a ⏎ - b" {
		t.Errorf("excerpt = %q", h.Excerpt)
	}
}
