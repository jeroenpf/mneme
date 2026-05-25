package api_test

import (
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestListProjectsWithCounts(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")
	seedProject(t, "hermes")

	mk := func(title, project, status string) {
		body := map[string]any{
			"meta": map[string]any{
				"title": title, "type": "plan", "project": project, "status": status,
			},
			"body": map[string]any{},
		}
		resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", body)
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}
	mk("A1", "apollo", "todo")
	mk("A2", "apollo", "in-progress")
	mk("A3", "apollo", "complete")
	mk("A4", "apollo", "archived")
	mk("H1", "hermes", "blocked")

	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects", nil)
	requireStatus(t, resp, http.StatusOK)

	var got struct {
		Items []models.ProjectStats `json:"items"`
	}
	decodeBody(t, resp, &got)

	if len(got.Items) != 2 {
		t.Fatalf("got %d projects, want 2", len(got.Items))
	}
	byslug := map[string]models.ProjectStats{}
	for _, p := range got.Items {
		byslug[p.Slug] = p
	}

	apollo := byslug["apollo"]
	if apollo.Counts.Total != 4 ||
		apollo.Counts.Todo != 1 ||
		apollo.Counts.InProgress != 1 ||
		apollo.Counts.Complete != 1 ||
		apollo.Counts.Archived != 1 {
		t.Errorf("apollo counts: %+v", apollo.Counts)
	}
	hermes := byslug["hermes"]
	if hermes.Counts.Total != 1 || hermes.Counts.Blocked != 1 {
		t.Errorf("hermes counts: %+v", hermes.Counts)
	}
}

func TestListProjectsEmpty(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects", nil)
	requireStatus(t, resp, http.StatusOK)

	var got struct {
		Items []models.ProjectStats `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 0 {
		t.Errorf("expected empty list, got %d items", len(got.Items))
	}
}
