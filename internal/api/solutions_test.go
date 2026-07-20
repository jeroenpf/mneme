package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
)

func TestSolutionListEmpty(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/solutions", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.Solution `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 0 {
		t.Fatalf("expected empty list, got %+v", got.Items)
	}
}

func TestSolutionListAndFilter(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()

	if err := st.CreateSolution(ctx, &models.Solution{
		ErrorDescription: "global gotcha", Solution: "global fix", Tags: []string{"macos"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSolution(ctx, &models.Solution{
		ErrorDescription: "apollo gotcha", Solution: "apollo fix",
		Project: strptr("apollo"), Tags: []string{"docker"},
	}); err != nil {
		t.Fatal(err)
	}

	// unfiltered -> both
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/solutions", nil)
	requireStatus(t, resp, http.StatusOK)
	var all struct {
		Items []models.Solution `json:"items"`
	}
	decodeBody(t, resp, &all)
	if len(all.Items) != 2 {
		t.Fatalf("unfiltered: want 2, got %d", len(all.Items))
	}

	// project filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/solutions?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var byProject struct {
		Items []models.Solution `json:"items"`
	}
	decodeBody(t, resp, &byProject)
	if len(byProject.Items) != 1 || byProject.Items[0].ErrorDescription != "apollo gotcha" {
		t.Fatalf("project filter: got %+v", byProject.Items)
	}

	// tag filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/solutions?tag=macos", nil)
	requireStatus(t, resp, http.StatusOK)
	var byTag struct {
		Items []models.Solution `json:"items"`
	}
	decodeBody(t, resp, &byTag)
	if len(byTag.Items) != 1 || byTag.Items[0].ErrorDescription != "global gotcha" {
		t.Fatalf("tag filter: got %+v", byTag.Items)
	}
}
