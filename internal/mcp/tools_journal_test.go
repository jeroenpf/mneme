package mcp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/store"
)

func TestAppendJournalReturnsID(t *testing.T) {
	cs := newClient(t)

	// Create acks with the public id alone — no echo of the payload.
	var created map[string]any
	call(t, cs, "append_journal", map[string]any{
		"summary": "built the thing", "accomplished": []string{"store", "api"},
	}, &created)
	pub, _ := created["public_id"].(string)
	if !strings.HasPrefix(pub, "jrnl_") {
		t.Fatalf("create ack public_id: got %v", created)
	}
	if len(created) != 1 {
		t.Errorf("create ack should be exactly {public_id}, got %v", created)
	}

	// Update accepts the public id and acks the same way.
	var updated map[string]any
	call(t, cs, "append_journal", map[string]any{"id": pub, "summary": "refined"}, &updated)
	if updated["public_id"] != pub {
		t.Errorf("update ack: got %v, want public_id %s", updated, pub)
	}
	if len(updated) != 1 {
		t.Errorf("update ack should be exactly {public_id}, got %v", updated)
	}
}

func TestAppendJournalCreate(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "append_journal", map[string]any{
		"project":      "apollo",
		"session_ref":  "sp-2-4",
		"summary":      "Built the journal store",
		"accomplished": []string{"migration", "store methods"},
		"deferred":     []string{"vue timeline"},
	}, &created)
	if created.PublicID == "" {
		t.Fatal("expected a public id")
	}

	e, err := store.NewWithPool(testPool).GetJournalEntryByPublicID(context.Background(), created.PublicID)
	if err != nil {
		t.Fatalf("load stored entry: %v", err)
	}
	if e.Summary != "Built the journal store" || len(e.Accomplished) != 2 {
		t.Errorf("stored entry: summary=%q accomplished=%v", e.Summary, e.Accomplished)
	}
}

func TestAppendJournalUnknownProject(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "append_journal", map[string]any{
		"project": "ghost", "summary": "s",
	})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestAppendJournalMissingSummary(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "append_journal", map[string]any{"session_ref": "sp-1-1"})
	if msg == "" {
		t.Error("expected missing-summary error")
	}
}

func TestAppendJournalUpsert(t *testing.T) {
	cs := newClient(t)
	var created struct {
		PublicID string `json:"public_id"`
	}
	call(t, cs, "append_journal", map[string]any{
		"session_ref": "sp-1-2", "summary": "draft", "deferred": []string{"tests"},
	}, &created)

	call(t, cs, "append_journal", map[string]any{
		"id": created.PublicID, "summary": "final", "deferred": []string{},
	}, nil)

	e, err := store.NewWithPool(testPool).GetJournalEntryByPublicID(context.Background(), created.PublicID)
	if err != nil {
		t.Fatalf("load stored entry: %v", err)
	}
	if e.Summary != "final" {
		t.Errorf("summary not updated: %q", e.Summary)
	}
	if e.SessionRef != "sp-1-2" { // partial update preserves session_ref
		t.Errorf("session_ref clobbered: %q", e.SessionRef)
	}
	if len(e.Deferred) != 0 { // explicit [] clears the list
		t.Errorf("deferred not cleared: %+v", e.Deferred)
	}
}

