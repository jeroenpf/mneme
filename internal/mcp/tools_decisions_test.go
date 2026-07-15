package mcp_test

import (
	"fmt"
	"testing"
)

func TestLogDecisionCreateAndGet(t *testing.T) {
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

	var listed struct {
		Decisions []struct {
			Title string `json:"title"`
		} `json:"decisions"`
	}
	call(t, cs, "get_decisions", map[string]any{"project": "apollo"}, &listed)
	if len(listed.Decisions) != 1 || listed.Decisions[0].Title != "Use pgx over database/sql" {
		t.Fatalf("get_decisions: %+v", listed.Decisions)
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

func TestQueryDecisions(t *testing.T) {
	cs := newClient(t)
	call(t, cs, "log_decision", map[string]any{
		"title": "Choose Sanctum for API auth", "decision": "Laravel Sanctum.",
		"rationale": "Token auth without OAuth ceremony.",
	}, nil)
	call(t, cs, "log_decision", map[string]any{
		"title": "Cursor pagination", "decision": "Keyset on id.",
		"rationale": "Stable pages under inserts.",
	}, nil)

	var out struct {
		Decisions []struct {
			Title string `json:"title"`
		} `json:"decisions"`
	}
	call(t, cs, "query_decisions", map[string]any{"query": "sanctum auth"}, &out)
	if len(out.Decisions) == 0 || out.Decisions[0].Title != "Choose Sanctum for API auth" {
		t.Fatalf("query_decisions ranking: %+v", out.Decisions)
	}
}

func TestGetDecisionsDefaultCap(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	for i := range 25 {
		call(t, cs, "log_decision", map[string]any{
			"title": fmt.Sprintf("D%d", i), "decision": "x", "project": "apollo",
		}, nil)
	}
	var out struct {
		Decisions []map[string]any `json:"decisions"`
	}
	call(t, cs, "get_decisions", map[string]any{}, &out)
	if len(out.Decisions) != 20 {
		t.Errorf("default cap: got %d decisions, want 20", len(out.Decisions))
	}
}
