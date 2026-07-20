package embed

import (
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/models"
)

func strptr(s string) *string { return &s }

func TestChunksDocumentPerSection(t *testing.T) {
	doc := &models.Document{
		ID: "d1", Title: "Zigbee plan", Project: strptr("apollo"),
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "Overview",
				"content": "migrate the coordinator"},
			map[string]any{"type": "section", "id": "risks", "title": "Risks",
				"content": "pairing may fail"},
		}},
	}
	got := Chunks(doc)
	if len(got) != 2 {
		t.Fatalf("expected 2 section chunks, got %d: %+v", len(got), got)
	}
	if got[0].ID != "overview" {
		t.Errorf("chunk id: got %q want overview", got[0].ID)
	}
	// chunk text carries title | project | section title : content
	for _, want := range []string{"Zigbee plan", "apollo", "Overview", "migrate the coordinator"} {
		if !strings.Contains(got[0].Text, want) {
			t.Errorf("chunk text %q missing %q", got[0].Text, want)
		}
	}
}

func TestChunksShortEntitySingle(t *testing.T) {
	dec := &models.Decision{ID: "x", Title: "Use zigbee2mqtt", Decision: "adopt it", Project: strptr("apollo")}
	got := Chunks(dec)
	if len(got) != 1 || got[0].ID != "full" {
		t.Fatalf("expected single 'full' chunk, got %+v", got)
	}
	if !strings.Contains(got[0].Text, "Use zigbee2mqtt") || !strings.Contains(got[0].Text, "adopt it") {
		t.Errorf("decision chunk text missing fields: %q", got[0].Text)
	}
}

func TestNewClientNilWithoutKey(t *testing.T) {
	if NewClient(config.Config{VoyageKey: ""}) != nil {
		t.Fatal("expected nil client without a key")
	}
	if NewClient(config.Config{VoyageKey: "k", VoyageModel: "voyage-4-large"}) == nil {
		t.Fatal("expected a client with a key")
	}
}
