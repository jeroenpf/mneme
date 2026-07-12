package mcp_test

import "testing"

func TestGetContextBundle(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	// seed content through the existing tools so the bundle has data
	call(t, cs, "set_memory", map[string]any{
		"scope": "project", "project": "apollo", "key": "db", "value": "postgres",
	}, nil)
	call(t, cs, "log_decision", map[string]any{
		"project": "apollo", "title": "use pgx", "decision": "pgx/v5",
	}, nil)

	var b struct {
		Project   string            `json:"project"`
		Memory    map[string]string `json:"memory"`
		Decisions []struct {
			Title string `json:"title"`
		} `json:"decisions"`
		Markdown string `json:"markdown"`
	}
	call(t, cs, "get_context_bundle", map[string]any{"project": "apollo"}, &b)
	if b.Project != "apollo" {
		t.Fatalf("project: %q", b.Project)
	}
	if b.Memory["db"] != "postgres" {
		t.Errorf("memory not assembled: %v", b.Memory)
	}
	if len(b.Decisions) != 1 || b.Decisions[0].Title != "use pgx" {
		t.Errorf("decisions: %+v", b.Decisions)
	}
	if b.Markdown == "" {
		t.Error("expected a markdown digest")
	}
}

func TestGetContextBundleUnknownProject(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "get_context_bundle", map[string]any{"project": "ghost"}); msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestGetContextBundleMissingProject(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "get_context_bundle", map[string]any{}); msg == "" {
		t.Error("expected project-required error")
	}
}
