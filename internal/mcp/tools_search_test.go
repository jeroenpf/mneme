package mcp_test

import "testing"

func TestSearchTool(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	// "zigbee2mqtt" alone tokenizes as a single numword that does NOT match
	// "zigbee"; the bare "zigbee" in the decision text is what the query
	// hits (discovered during execution — see the plan's note). Title stays
	// "use zigbee2mqtt" so the assertion below is unaffected.
	call(t, cs, "log_decision", map[string]any{
		"project": "apollo", "title": "use zigbee2mqtt", "decision": "adopt zigbee2mqtt for the zigbee mesh",
	}, nil)

	var out struct {
		Results []struct {
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"results"`
	}
	call(t, cs, "search", map[string]any{"q": "zigbee"}, &out)
	found := false
	for _, r := range out.Results {
		if r.Type == "decisions" && r.Title == "use zigbee2mqtt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected decision hit, got %+v", out.Results)
	}
}

func TestSearchToolTypeFilter(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "log_decision", map[string]any{
		"project": "apollo", "title": "zigbee choice", "decision": "zigbee",
	}, nil)

	var out struct {
		Results []struct {
			Type string `json:"type"`
		} `json:"results"`
	}
	call(t, cs, "search", map[string]any{"q": "zigbee", "types": []string{"snippets"}}, &out)
	for _, r := range out.Results {
		if r.Type != "snippets" {
			t.Errorf("types filter leaked: %s", r.Type)
		}
	}
}

func TestSearchToolEmptyHint(t *testing.T) {
	// With embedding active (fake client) and an empty corpus, a query that
	// matches nothing returns no results plus a rephrase hint so the LLM
	// caller knows to refine — not just an ambiguous empty list.
	cs := newClientWith(t, fakeEmbedClient{})
	var out struct {
		Results []any  `json:"results"`
		Note    string `json:"note"`
	}
	call(t, cs, "search", map[string]any{"q": "nothingmatchesthisquery"}, &out)
	if len(out.Results) != 0 {
		t.Fatalf("expected no results, got %+v", out.Results)
	}
	if out.Note == "" {
		t.Fatal("expected a hint note when a hybrid search returns nothing")
	}
}

func TestSearchToolMissingQuery(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "search", map[string]any{}); msg == "" {
		t.Error("expected query-required error")
	}
}

func TestSearchToolUnknownType(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "search", map[string]any{"q": "x", "types": []string{"bogus"}}); msg == "" {
		t.Error("expected unknown-type error")
	}
}
