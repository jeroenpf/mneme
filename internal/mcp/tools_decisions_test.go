package mcp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/store"
)

func TestLogDecisionReturnsID(t *testing.T) {
	cs := newClient(t)

	// Create acks with the public id alone — no echo of the payload.
	var created map[string]any
	call(t, cs, "log_decision", map[string]any{
		"title": "Use pgx", "decision": "pgx/v5", "alternatives": "database/sql",
	}, &created)
	pub, _ := created["public_id"].(string)
	if !strings.HasPrefix(pub, "dec_") {
		t.Fatalf("create ack public_id: got %v", created)
	}
	if len(created) != 1 {
		t.Errorf("create ack should be exactly {public_id}, got %v", created)
	}

	// Update accepts the public id and acks the same way.
	var updated map[string]any
	call(t, cs, "log_decision", map[string]any{"id": pub, "status": "deprecated"}, &updated)
	if updated["public_id"] != pub {
		t.Errorf("update ack: got %v, want public_id %s", updated, pub)
	}
	if len(updated) != 1 {
		t.Errorf("update ack should be exactly {public_id}, got %v", updated)
	}
}

func TestLogDecisionCreate(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "log_decision", map[string]any{
		"title":     "Use pgx over database/sql",
		"project":   "apollo",
		"decision":  "Adopt jackc/pgx v5.",
		"rationale": "Native Postgres types.",
	}, &created)
	if created.PublicID == "" {
		t.Fatal("expected a public id")
	}

	d, err := store.NewWithPool(testPool).GetDecisionByPublicID(context.Background(), created.PublicID)
	if err != nil {
		t.Fatalf("load stored decision: %v", err)
	}
	if d.Title != "Use pgx over database/sql" || string(d.Status) != "accepted" {
		t.Errorf("stored decision: title=%q status=%q", d.Title, d.Status)
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
		PublicID string `json:"public_id"`
	}
	call(t, cs, "log_decision", map[string]any{
		"title": "Presence detection", "decision": "mmWave.", "status": "proposed",
	}, &created)

	call(t, cs, "log_decision", map[string]any{"id": created.PublicID, "status": "deprecated"}, nil)

	d, err := store.NewWithPool(testPool).GetDecisionByPublicID(context.Background(), created.PublicID)
	if err != nil {
		t.Fatalf("load stored decision: %v", err)
	}
	if string(d.Status) != "deprecated" {
		t.Errorf("status not updated: %q", d.Status)
	}
	if d.Title != "Presence detection" { // partial update preserves title
		t.Errorf("title clobbered: %q", d.Title)
	}
}

