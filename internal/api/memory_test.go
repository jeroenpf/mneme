package api_test

import (
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestMemoryPutAndGet(t *testing.T) {
	srv, _ := newServer(t)

	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/global/editor", map[string]any{"value": "neovim"})
	requireStatus(t, resp, http.StatusOK)
	var put models.Memory
	decodeBody(t, resp, &put)
	if put.Key != "editor" || put.Value != "neovim" || put.Scope != models.ScopeGlobal {
		t.Fatalf("put entry: %+v", put)
	}

	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/memory?scope=global", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.Memory `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 1 || got.Items[0].Value != "neovim" {
		t.Fatalf("get items: %+v", got.Items)
	}
}

func TestMemoryProjectScopeRequiresProject(t *testing.T) {
	srv, _ := newServer(t)
	// unknown project -> 400 (FK -> ErrInvalidProject)
	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/project/stack?project=ghost", map[string]any{"value": "go"})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()

	seedProject(t, "apollo")
	resp = doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/project/stack?project=apollo", map[string]any{"value": "go"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestMemoryScopeShapeValidation(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")
	// global scope must not carry a project
	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/global/x?project=apollo", map[string]any{"value": "v"})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
	// area scope requires area
	resp = doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/area/x?project=apollo", map[string]any{"value": "v"})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestMemoryDelete(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/memory/global/k", map[string]any{"value": "v"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	del := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/memory/global/k", nil)
	requireStatus(t, del, http.StatusNoContent)
	del.Body.Close()

	del2 := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/memory/global/k", nil)
	requireStatus(t, del2, http.StatusNotFound)
	del2.Body.Close()
}
