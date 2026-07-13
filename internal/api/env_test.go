package api_test

import (
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestEnvPutListDelete(t *testing.T) {
	srv, _ := newServer(t)
	seedProject(t, "apollo")

	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/env/API_PORT?project=apollo",
		map[string]any{"value": "8443", "description": "https port"})
	requireStatus(t, resp, http.StatusOK)
	var put models.EnvEntry
	decodeBody(t, resp, &put)
	if put.Key != "API_PORT" || put.Value != "8443" || put.Description == nil || *put.Description != "https port" {
		t.Fatalf("put entry: %+v", put)
	}

	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/env?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.EnvEntry `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 1 || got.Items[0].Value != "8443" {
		t.Fatalf("get items: %+v", got.Items)
	}

	del := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/env/API_PORT?project=apollo", nil)
	requireStatus(t, del, http.StatusNoContent)
	del.Body.Close()

	del2 := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/env/API_PORT?project=apollo", nil)
	requireStatus(t, del2, http.StatusNotFound)
	del2.Body.Close()
}

func TestEnvListRequiresProject(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/env", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestEnvUnknownProject(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodPut, srv.URL+"/api/v1/env/K?project=ghost",
		map[string]any{"value": "v"})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}
