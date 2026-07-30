package mcp_test

import "testing"

func TestLogDecisionCreate(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	call(t, cs, "log_decision", map[string]any{
		"title":     "Use pgx over database/sql",
		"project":   "apollo",
		"decision":  "Adopt jackc/pgx v5.",
		"rationale": "Native Postgres types.",
	}, &created)
	if created.ID == "" {
		t.Fatal("expected a generated id")
	}
	if created.Status != "accepted" { // default
		t.Errorf("status: got %q, want accepted", created.Status)
	}
}

func TestLogDecisionUnknownProject(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "log_decision", map[string]any{
		"title": "x", "project": "ghost", "decision": "d",
	})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestLogDecisionMissingTitle(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "log_decision", map[string]any{"decision": "d"})
	if msg == "" {
		t.Error("expected missing-title error")
	}
}

func TestLogDecisionUpsertStatus(t *testing.T) {
	cs := newClient(t)
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	call(t, cs, "log_decision", map[string]any{
		"title": "Presence detection", "decision": "mmWave.", "status": "proposed",
	}, &created)
	if created.Status != "proposed" {
		t.Fatalf("initial status: %q", created.Status)
	}

	var updated struct {
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	call(t, cs, "log_decision", map[string]any{"id": created.ID, "status": "deprecated"}, &updated)
	if updated.Status != "deprecated" {
		t.Errorf("status not updated: %q", updated.Status)
	}
	if updated.Title != "Presence detection" { // partial update preserves title
		t.Errorf("title clobbered: %q", updated.Title)
	}
}

