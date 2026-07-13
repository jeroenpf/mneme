package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestSearchMissingQuery(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/search", nil)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestSearchUnknownType(t *testing.T) {
	srv, _ := newServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/search?q=zigbee&types=bogus", nil)
	requireStatus(t, resp, http.StatusBadRequest)
}

func TestSearchReturnsHits(t *testing.T) {
	srv, st := newServer(t)
	seedProject(t, "apollo")
	ctx := context.Background()
	// The bare "zigbee" in the decision text is what the query hits;
	// "zigbee2mqtt" alone tokenizes as a single numword that does NOT match
	// "zigbee" (discovered during execution — see the plan's note).
	if err := st.CreateDecision(ctx, &models.Decision{
		Title: "use zigbee2mqtt", Project: strptr("apollo"),
		Decision: "adopt zigbee2mqtt for the zigbee mesh", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}

	resp := doJSON(t, http.MethodGet, srv.URL+"/api/v1/search?q=zigbee", nil)
	requireStatus(t, resp, http.StatusOK)
	var body struct {
		Items []struct {
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) == 0 || body.Items[0].Title == "" {
		t.Fatalf("expected hits, got %+v", body)
	}
}
