package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestSearchStatus(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()
	if err := st.CreateDecision(ctx, &models.Decision{
		Title: "t", Project: strptr("apollo"), Decision: "d", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}

	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/search/status", nil)
	requireStatus(t, resp, http.StatusOK)
	var body struct {
		Enabled bool `json:"enabled"`
		Items   []struct {
			Type     string `json:"type"`
			Embedded int    `json:"embedded"`
			Total    int    `json:"total"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	byType := map[string]int{}
	for _, it := range body.Items {
		byType[it.Type] = it.Total
	}
	if byType["decisions"] < 1 {
		t.Fatalf("expected decisions total >= 1, got %+v", body.Items)
	}
}
