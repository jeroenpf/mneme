package api_test

import (
	"net/http"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
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

func TestCreateProject(t *testing.T) {
	srv, _ := newServer(t)

	body := map[string]any{"slug": "tradegod", "name": "TradeGod", "description": "Trading bot"}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", body)
	requireStatus(t, resp, http.StatusCreated)

	var got models.Project
	decodeBody(t, resp, &got)
	if got.Slug != "tradegod" || got.Name != "TradeGod" {
		t.Fatalf("got %+v, want tradegod/TradeGod", got)
	}
	if got.ID == "" || got.CreatedAt.IsZero() {
		t.Errorf("id/created_at not populated: %+v", got)
	}
	if got.Description == nil || *got.Description != "Trading bot" {
		t.Errorf("description: got %v", got.Description)
	}

	// A document may now reference the freshly created project.
	docBody := map[string]any{
		"meta": map[string]any{"title": "First", "type": "plan", "project": "tradegod"},
		"body": map[string]any{},
	}
	dresp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/documents", docBody)
	requireStatus(t, dresp, http.StatusCreated)
	dresp.Body.Close()
}

func TestCreateProjectNormalizesSlug(t *testing.T) {
	srv, _ := newServer(t)

	body := map[string]any{"slug": "TradeGod Bot!", "name": "TradeGod"}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", body)
	requireStatus(t, resp, http.StatusCreated)

	var got models.Project
	decodeBody(t, resp, &got)
	if got.Slug != "tradegod-bot" {
		t.Errorf("slug not normalized: got %q, want %q", got.Slug, "tradegod-bot")
	}
}

func TestCreateProjectDuplicate(t *testing.T) {
	srv, _ := newServer(t)
	body := map[string]any{"slug": "dup", "name": "Dup"}

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", body)
	requireStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp2 := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", body)
	requireStatus(t, resp2, http.StatusConflict)
	resp2.Body.Close()
}

func TestCreateProjectRequiresNameAndSlug(t *testing.T) {
	srv, _ := newServer(t)

	noName := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", map[string]any{"slug": "x"})
	requireStatus(t, noName, http.StatusBadRequest)
	noName.Body.Close()

	noSlug := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", map[string]any{"name": "X"})
	requireStatus(t, noSlug, http.StatusBadRequest)
	noSlug.Body.Close()
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
