package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestBundleMissingProject(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/bundle", nil)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestBundleUnknownProject(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/bundle?project=ghost", nil)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestBundleAssembled(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()
	if err := st.SetMemory(ctx, &models.Memory{
		Scope: models.ScopeProject, Project: strptr("apollo"), Key: "db", Value: "postgres",
	}); err != nil {
		t.Fatal(err)
	}

	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/bundle?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var b struct {
		Project  string            `json:"project"`
		Memory   map[string]string `json:"memory"`
		Markdown string            `json:"markdown"`
	}
	decodeBody(t, resp, &b)
	if b.Project != "apollo" || b.Memory["db"] != "postgres" || b.Markdown == "" {
		t.Fatalf("bundle: %+v", b)
	}
}
