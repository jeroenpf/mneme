package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
)

// createPlan POSTs a plan with the given body and returns the created document.
func createPlan(t *testing.T, srvURL, title string, body map[string]any) models.Document {
	t.Helper()
	resp := doJSON(t, http.MethodPost, srvURL+"/api/v1/documents", map[string]any{
		"meta": map[string]any{"title": title, "type": "plan", "project": "apollo"},
		"body": body,
	})
	requireStatus(t, resp, http.StatusCreated)
	var created models.Document
	decodeBody(t, resp, &created)
	return created
}

func sectionBody(id, title string) map[string]any {
	return map[string]any{"sections": []any{
		map[string]any{"type": "section", "id": id, "title": title, "content": "x"},
	}}
}

// GET /documents/{id}/revisions lists compact, newest-first revision summaries
// (no body/meta) with op, actor, and the ids each write touched.
func TestListDocumentRevisionsREST(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")

	created := createPlan(t, srv.URL, "Hist Doc", sectionBody("s1", "A"))
	patch := doPatchIfMatch(t, srv.URL+"/api/v1/documents/"+created.ID, `{"status":"in-progress"}`, "")
	requireStatus(t, patch, http.StatusOK)
	patch.Body.Close()

	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents/"+created.ID+"/revisions", nil)
	requireStatus(t, resp, http.StatusOK)
	var body struct {
		Items []map[string]any `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 2 {
		t.Fatalf("revisions len = %d, want 2 (create + patch)", len(body.Items))
	}
	// Newest first.
	if body.Items[0]["op"] != "rest:update" || body.Items[1]["op"] != "rest:create" {
		t.Fatalf("ops = %v, %v; want rest:update, rest:create", body.Items[0]["op"], body.Items[1]["op"])
	}
	if body.Items[0]["actor"] != "rest" {
		t.Fatalf("actor = %v, want rest", body.Items[0]["actor"])
	}
	// Summaries are compact — no body/meta payload.
	if _, ok := body.Items[0]["body"]; ok {
		t.Fatal("revision summary should not include the body")
	}
	if _, ok := body.Items[0]["revision"]; !ok {
		t.Fatal("revision summary should include the revision number")
	}
}

// POST /documents/{id}/restore rewinds content to a past revision by writing a
// new forward revision (history is append-only).
func TestRestoreDocumentRevisionREST(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")

	created := createPlan(t, srv.URL, "Restore Doc", sectionBody("s1", "ORIGINAL"))
	// Update the body so revision 1 differs from the current state.
	patch := doPatchIfMatch(t, srv.URL+"/api/v1/documents/"+created.ID,
		`{"body":{"sections":[{"type":"section","id":"s1","title":"CHANGED","content":"x"}]}}`, "")
	requireStatus(t, patch, http.StatusOK)
	patch.Body.Close()

	// Restore revision 1.
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents/"+created.ID+"/restore",
		map[string]any{"revision": 1})
	requireStatus(t, resp, http.StatusOK)
	var out struct {
		RestoredFrom int              `json:"restored_from"`
		NewRevision  int              `json:"new_revision"`
		Doc          *models.Document `json:"doc"`
	}
	decodeBody(t, resp, &out)
	if out.RestoredFrom != 1 {
		t.Fatalf("restored_from = %d, want 1", out.RestoredFrom)
	}
	if out.NewRevision != 3 {
		t.Fatalf("new_revision = %d, want 3 (create=1, patch=2, restore=3)", out.NewRevision)
	}
	// The restored doc's body must match revision 1's content again.
	raw, _ := json.Marshal(out.Doc.Body)
	if got := string(raw); !strings.Contains(got, "ORIGINAL") || strings.Contains(got, "CHANGED") {
		t.Fatalf("restored body did not rewind to revision 1: %s", got)
	}
}

func TestRestoreMissingRevisionREST(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")
	created := createPlan(t, srv.URL, "No Such Rev", sectionBody("s1", "A"))

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents/"+created.ID+"/restore",
		map[string]any{"revision": 99})
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestRestoreBadRevisionREST(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")
	created := createPlan(t, srv.URL, "Bad Rev", sectionBody("s1", "A"))

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents/"+created.ID+"/restore",
		map[string]any{"revision": 0})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}
