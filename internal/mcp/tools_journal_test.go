package mcp_test

import "testing"

func TestAppendJournalCreate(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	}
	call(t, cs, "append_journal", map[string]any{
		"project":      "apollo",
		"session_ref":  "sp-2-4",
		"summary":      "Built the journal store",
		"accomplished": []string{"migration", "store methods"},
		"deferred":     []string{"vue timeline"},
	}, &created)
	if created.ID == "" {
		t.Fatal("expected a generated id")
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
		ID string `json:"id"`
	}
	call(t, cs, "append_journal", map[string]any{
		"session_ref": "sp-1-2", "summary": "draft", "deferred": []string{"tests"},
	}, &created)

	var updated struct {
		SessionRef string   `json:"session_ref"`
		Summary    string   `json:"summary"`
		Deferred   []string `json:"deferred"`
	}
	call(t, cs, "append_journal", map[string]any{
		"id": created.ID, "summary": "final", "deferred": []string{},
	}, &updated)
	if updated.Summary != "final" {
		t.Errorf("summary not updated: %q", updated.Summary)
	}
	if updated.SessionRef != "sp-1-2" { // partial update preserves session_ref
		t.Errorf("session_ref clobbered: %q", updated.SessionRef)
	}
	if len(updated.Deferred) != 0 { // explicit [] clears the list
		t.Errorf("deferred not cleared: %+v", updated.Deferred)
	}
}

