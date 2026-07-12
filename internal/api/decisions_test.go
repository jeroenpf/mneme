package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestDecisionsListEmpty(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/decisions", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.Decision `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 0 {
		t.Fatalf("expected empty list, got %+v", got.Items)
	}
}

func TestDecisionsListAndFilter(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()

	if err := st.CreateDecision(ctx, &models.Decision{
		Title: "global one", Decision: "d", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDecision(ctx, &models.Decision{
		Title: "apollo proposed", Project: strptr("apollo"), Decision: "d", Status: models.DecisionProposed,
	}); err != nil {
		t.Fatal(err)
	}

	// unfiltered -> both
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/decisions", nil)
	requireStatus(t, resp, http.StatusOK)
	var all struct {
		Items []models.Decision `json:"items"`
	}
	decodeBody(t, resp, &all)
	if len(all.Items) != 2 {
		t.Fatalf("unfiltered: want 2, got %d", len(all.Items))
	}

	// project filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/decisions?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var byProject struct {
		Items []models.Decision `json:"items"`
	}
	decodeBody(t, resp, &byProject)
	if len(byProject.Items) != 1 || byProject.Items[0].Title != "apollo proposed" {
		t.Fatalf("project filter: got %+v", byProject.Items)
	}

	// status filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/decisions?status=proposed", nil)
	requireStatus(t, resp, http.StatusOK)
	var byStatus struct {
		Items []models.Decision `json:"items"`
	}
	decodeBody(t, resp, &byStatus)
	if len(byStatus.Items) != 1 || byStatus.Items[0].Status != models.DecisionProposed {
		t.Fatalf("status filter: got %+v", byStatus.Items)
	}
}

func TestDecisionsInvalidStatus(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/decisions?status=bogus", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func strptr(s string) *string { return &s }
