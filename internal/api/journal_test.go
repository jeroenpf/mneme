package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestJournalListEmpty(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal", nil)
	requireStatus(t, resp, http.StatusOK)
	var got struct {
		Items []models.JournalEntry `json:"items"`
	}
	decodeBody(t, resp, &got)
	if len(got.Items) != 0 {
		t.Fatalf("expected empty list, got %+v", got.Items)
	}
}

func TestJournalListAndFilter(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()

	if err := st.CreateJournalEntry(ctx, &models.JournalEntry{
		Summary: "global session", Accomplished: []string{"x"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateJournalEntry(ctx, &models.JournalEntry{
		Summary: "apollo session", Project: strptr("apollo"), SessionRef: "sp-1-1",
	}); err != nil {
		t.Fatal(err)
	}

	// unfiltered -> both
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal", nil)
	requireStatus(t, resp, http.StatusOK)
	var all struct {
		Items []models.JournalEntry `json:"items"`
	}
	decodeBody(t, resp, &all)
	if len(all.Items) != 2 {
		t.Fatalf("unfiltered: want 2, got %d", len(all.Items))
	}

	// project filter
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal?project=apollo", nil)
	requireStatus(t, resp, http.StatusOK)
	var byProject struct {
		Items []models.JournalEntry `json:"items"`
	}
	decodeBody(t, resp, &byProject)
	if len(byProject.Items) != 1 || byProject.Items[0].Summary != "apollo session" {
		t.Fatalf("project filter: got %+v", byProject.Items)
	}

	// since filter: a clearly-past date returns all; a clearly-future date returns none
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal?since=2020-01-01", nil)
	requireStatus(t, resp, http.StatusOK)
	var sincePast struct {
		Items []models.JournalEntry `json:"items"`
	}
	decodeBody(t, resp, &sincePast)
	if len(sincePast.Items) != 2 {
		t.Fatalf("since=past: want 2, got %d", len(sincePast.Items))
	}

	resp = doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal?since=2099-01-01", nil)
	requireStatus(t, resp, http.StatusOK)
	var sinceFuture struct {
		Items []models.JournalEntry `json:"items"`
	}
	decodeBody(t, resp, &sinceFuture)
	if len(sinceFuture.Items) != 0 {
		t.Fatalf("since=future: want 0, got %d", len(sinceFuture.Items))
	}
}

func TestJournalInvalidSince(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/journal?since=not-a-date", nil)
	requireStatus(t, resp, http.StatusBadRequest)
}
