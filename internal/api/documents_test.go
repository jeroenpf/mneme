package api_test

import (
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestCreateAndGetDocument(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")

	body := map[string]any{
		"meta": map[string]any{
			"title":   "Vehicle Listing API",
			"type":    "plan",
			"project": "apollo",
			"ticket":  "C1-142",
			"tags":    []string{"go", "api"},
		},
		"body": map[string]any{
			"sections": []any{
				map[string]any{"type": "section", "id": "overview"},
			},
		},
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", body)
	requireStatus(t, resp, http.StatusCreated)
	var created models.Document
	decodeBody(t, resp, &created)

	if created.ID != "vehicle-listing-api" {
		t.Errorf("id: got %q, want %q", created.ID, "vehicle-listing-api")
	}
	if created.Status != models.StatusTodo {
		t.Errorf("default status: got %q, want %q", created.Status, models.StatusTodo)
	}
	if created.Ticket == nil || *created.Ticket != "C1-142" {
		t.Errorf("ticket not promoted: %+v", created.Ticket)
	}

	// Round-trip via GET.
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents/"+created.ID, nil)
	requireStatus(t, resp, http.StatusOK)
	var fetched models.Document
	decodeBody(t, resp, &fetched)
	if fetched.Title != "Vehicle Listing API" {
		t.Errorf("fetched title: %q", fetched.Title)
	}
	if len(fetched.Tags) != 2 {
		t.Errorf("tags not roundtripped: %v", fetched.Tags)
	}
}

func TestCreateDocumentSlugDedup(t *testing.T) {
	srv, _ := newServer(t)

	mk := func(title string) string {
		body := map[string]any{
			"meta": map[string]any{"title": title, "type": "plan"},
			"body": map[string]any{},
		}
		resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", body)
		requireStatus(t, resp, http.StatusCreated)
		var d models.Document
		decodeBody(t, resp, &d)
		return d.ID
	}

	id1 := mk("Same Title")
	id2 := mk("Same Title")
	id3 := mk("Same Title")
	if id1 != "same-title" || id2 != "same-title-2" || id3 != "same-title-3" {
		t.Errorf("dedup ids: got %q, %q, %q", id1, id2, id3)
	}
}

func TestCreateDocumentValidation(t *testing.T) {
	srv, _ := newServer(t)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing meta", map[string]any{"body": map[string]any{}}, http.StatusBadRequest},
		{"missing title", map[string]any{
			"meta": map[string]any{"type": "plan"},
		}, http.StatusBadRequest},
		{"missing type", map[string]any{
			"meta": map[string]any{"title": "X"},
		}, http.StatusBadRequest},
		{"unknown project", map[string]any{
			"meta": map[string]any{"title": "X", "type": "plan", "project": "ghost"},
		}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", tc.body)
			requireStatus(t, resp, tc.want)
			resp.Body.Close()
		})
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents/nope", nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListDocumentsFiltersAndSearch(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")
	seedProject(t, "hermes")

	mk := func(title, typ, project string, tags []string) {
		body := map[string]any{
			"meta": map[string]any{
				"title": title, "type": typ, "project": project, "tags": tags,
			},
			"body": map[string]any{},
		}
		resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", body)
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}
	mk("Vehicle Listing API", "plan", "apollo", []string{"go", "api"})
	mk("Inventory Spec", "spec", "apollo", []string{"go"})
	mk("Pricing Engine", "plan", "hermes", []string{"billing"})

	type listResp struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"next_cursor"`
	}

	// Filter by project.
	var r listResp
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &r)
	if len(r.Items) != 2 {
		t.Errorf("project=apollo: got %d items, want 2", len(r.Items))
	}

	// Filter by tags (CSV, must contain all).
	r = listResp{}
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?tags=go,api", nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &r)
	if len(r.Items) != 1 {
		t.Errorf("tags=go,api: got %d items, want 1", len(r.Items))
	}

	// Search (FTS).
	r = listResp{}
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?q=vehicle", nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &r)
	if len(r.Items) != 1 || r.Items[0]["id"] != "vehicle-listing-api" {
		t.Errorf("search vehicle: got %+v", r.Items)
	}
}

func TestListDocumentsCursorPagination(t *testing.T) {
	srv, _ := newServer(t)

	for i := 0; i < 5; i++ {
		body := map[string]any{
			"meta": map[string]any{"title": "Doc " + string(rune('A'+i)), "type": "plan"},
			"body": map[string]any{},
		}
		resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", body)
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	type listResp struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"next_cursor"`
	}

	// Page 1.
	var p1 listResp
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?limit=2", nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &p1)
	if len(p1.Items) != 2 {
		t.Fatalf("page1: got %d items, want 2", len(p1.Items))
	}
	if p1.NextCursor == nil {
		t.Fatalf("page1: expected next_cursor, got nil")
	}

	// Page 2.
	var p2 listResp
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?limit=2&cursor="+*p1.NextCursor, nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &p2)
	if len(p2.Items) != 2 {
		t.Fatalf("page2: got %d items, want 2", len(p2.Items))
	}
	if p2.NextCursor == nil {
		t.Fatalf("page2: expected next_cursor, got nil (5 total, 4 served)")
	}

	// Page 3 — last item, no next cursor.
	var p3 listResp
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents?limit=2&cursor="+*p2.NextCursor, nil)
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &p3)
	if len(p3.Items) != 1 {
		t.Fatalf("page3: got %d items, want 1", len(p3.Items))
	}
	if p3.NextCursor != nil {
		t.Errorf("page3: expected nil next_cursor, got %q", *p3.NextCursor)
	}

	// Pages must not overlap.
	seen := map[string]bool{}
	for _, items := range [][]map[string]any{p1.Items, p2.Items, p3.Items} {
		for _, it := range items {
			id := it["id"].(string)
			if seen[id] {
				t.Errorf("cursor pages overlap on %q", id)
			}
			seen[id] = true
		}
	}
}

func TestPatchDocumentPartial(t *testing.T) {
	srv, _ := newServer(t)

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", map[string]any{
		"meta": map[string]any{
			"title": "Original",
			"type":  "plan",
			"tags":  []string{"go"},
		},
		"body": map[string]any{"sections": []any{}},
	})
	requireStatus(t, resp, http.StatusCreated)
	var created models.Document
	decodeBody(t, resp, &created)

	// Patch only status — other fields must be preserved.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/api/v1/documents/"+created.ID, map[string]any{
		"status": "in-progress",
	})
	requireStatus(t, resp, http.StatusOK)
	var patched models.Document
	decodeBody(t, resp, &patched)
	if patched.Status != "in-progress" {
		t.Errorf("status: got %q, want in-progress", patched.Status)
	}
	if patched.Title != "Original" {
		t.Errorf("title clobbered: %q", patched.Title)
	}
	if len(patched.Tags) != 1 || patched.Tags[0] != "go" {
		t.Errorf("tags clobbered: %v", patched.Tags)
	}

	// Patch body — replaces full body, status preserved.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/api/v1/documents/"+created.ID, map[string]any{
		"body": map[string]any{"sections": []any{map[string]any{"id": "new"}}},
	})
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &patched)
	if patched.Status != "in-progress" {
		t.Errorf("status lost on body patch: %q", patched.Status)
	}
	secs, _ := patched.Body["sections"].([]any)
	if len(secs) != 1 {
		t.Errorf("body not replaced: %+v", patched.Body)
	}

	// Patch meta — replaces meta and re-promotes typed columns.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/api/v1/documents/"+created.ID, map[string]any{
		"meta": map[string]any{"title": "Renamed", "type": "plan", "tags": []string{"vue"}},
	})
	requireStatus(t, resp, http.StatusOK)
	decodeBody(t, resp, &patched)
	if patched.Title != "Renamed" {
		t.Errorf("rename failed: %q", patched.Title)
	}
	if len(patched.Tags) != 1 || patched.Tags[0] != "vue" {
		t.Errorf("tags not replaced: %v", patched.Tags)
	}
}

func TestPatchDocumentNotFound(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodPatch, srv.URL+"/api/v1/documents/ghost", map[string]any{
		"status": "complete",
	})
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestArchiveDocument(t *testing.T) {
	srv, _ := newServer(t)

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", map[string]any{
		"meta": map[string]any{"title": "To Be Archived", "type": "plan"},
		"body": map[string]any{},
	})
	requireStatus(t, resp, http.StatusCreated)
	var created models.Document
	decodeBody(t, resp, &created)

	resp = doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents/"+created.ID+"/archive", nil)
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Confirm state via GET.
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/documents/"+created.ID, nil)
	requireStatus(t, resp, http.StatusOK)
	var got models.Document
	decodeBody(t, resp, &got)
	if got.Status != models.StatusArchived {
		t.Errorf("status: got %q, want archived", got.Status)
	}

	// Archiving an unknown id is 404.
	resp = doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents/ghost/archive", nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}
