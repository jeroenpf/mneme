package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestSnippetsListEmpty(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/snippets", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.Snippet `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 0 {
		t.Fatalf("expected empty list, got %+v", got.Items)
	}
}

func TestSnippetsListAndFilter(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()

	if err := st.CreateSnippet(ctx, &models.Snippet{
		Title: "global go", Language: "go", Content: "c", Tags: []string{"util"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSnippet(ctx, &models.Snippet{
		Title: "apollo sql", Project: strptr("apollo"), Language: "sql", Content: "c", Tags: []string{"query"},
	}); err != nil {
		t.Fatal(err)
	}

	// unfiltered -> both
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/snippets", nil)
	requireStatus(t, resp, http.StatusOK)
	var all struct {
		Items []models.Snippet `json:"items"`
	}
	decodeBody(t, resp, &all)
	if len(all.Items) != 2 {
		t.Fatalf("unfiltered: want 2, got %d", len(all.Items))
	}

	// project filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/snippets?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var byProject struct {
		Items []models.Snippet `json:"items"`
	}
	decodeBody(t, resp, &byProject)
	if len(byProject.Items) != 1 || byProject.Items[0].Title != "apollo sql" {
		t.Fatalf("project filter: got %+v", byProject.Items)
	}

	// language filter (case-insensitive: ?language=GO matches stored "go")
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/snippets?language=GO", nil)
	requireStatus(t, resp, http.StatusOK)
	var byLang struct {
		Items []models.Snippet `json:"items"`
	}
	decodeBody(t, resp, &byLang)
	if len(byLang.Items) != 1 || byLang.Items[0].Title != "global go" {
		t.Fatalf("language filter: got %+v", byLang.Items)
	}

	// tag filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/snippets?tag=query", nil)
	requireStatus(t, resp, http.StatusOK)
	var byTag struct {
		Items []models.Snippet `json:"items"`
	}
	decodeBody(t, resp, &byTag)
	if len(byTag.Items) != 1 || byTag.Items[0].Title != "apollo sql" {
		t.Fatalf("tag filter: got %+v", byTag.Items)
	}
}
