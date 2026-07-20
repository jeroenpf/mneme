package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
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
			Type       string `json:"type"`
			Total      int    `json:"total"`
			Embedded   int    `json:"embedded"`
			Reconciled int    `json:"reconciled"`
			Missing    int    `json:"missing"`
			Stale      int    `json:"stale"`
			Orphaned   int    `json:"orphaned"`
			Failed     int    `json:"failed"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	var dec *int
	for i, it := range body.Items {
		if it.Type == "decisions" {
			// The seeded decision has no embedding → it's a missing source.
			if it.Total < 1 || it.Missing < 1 || it.Embedded != 0 {
				t.Fatalf("decisions buckets wrong: %+v", body.Items[i])
			}
			if it.Failed != 0 {
				t.Fatalf("no failures recorded, failed should be 0, got %d", it.Failed)
			}
			dec = &body.Items[i].Total
		}
	}
	if dec == nil {
		t.Fatalf("expected a decisions status item, got %+v", body.Items)
	}
}
