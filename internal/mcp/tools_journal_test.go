package mcp_test

import "testing"

func TestAppendJournalCreateAndGet(t *testing.T) {
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

	var listed struct {
		Entries []struct {
			Summary string `json:"summary"`
		} `json:"entries"`
	}
	call(t, cs, "get_journal", map[string]any{"project": "apollo"}, &listed)
	if len(listed.Entries) != 1 || listed.Entries[0].Summary != "Built the journal store" {
		t.Fatalf("get_journal: %+v", listed.Entries)
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

func TestGetJournalSince(t *testing.T) {
	cs := newClient(t)
	call(t, cs, "append_journal", map[string]any{"summary": "an entry"}, nil)

	var past struct {
		Entries []struct {
			Summary string `json:"summary"`
		} `json:"entries"`
	}
	call(t, cs, "get_journal", map[string]any{"since": "2020-01-01"}, &past)
	if len(past.Entries) != 1 {
		t.Fatalf("since=past should include the entry, got %+v", past.Entries)
	}

	var future struct {
		Entries []struct {
			Summary string `json:"summary"`
		} `json:"entries"`
	}
	call(t, cs, "get_journal", map[string]any{"since": "2099-01-01"}, &future)
	if len(future.Entries) != 0 {
		t.Fatalf("since=future should exclude the entry, got %+v", future.Entries)
	}
}
