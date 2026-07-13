package docmeta

import (
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

func TestFromMetaValidStatusAndType(t *testing.T) {
	doc, err := FromMeta(map[string]any{
		"title":  "Impl plan",
		"type":   "plan",
		"status": "in-progress",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Type != models.TypePlan || doc.Status != models.StatusInProgress {
		t.Fatalf("promoted wrong: type=%q status=%q", doc.Type, doc.Status)
	}
}

func TestFromMetaDefaultsStatusTodo(t *testing.T) {
	doc, err := FromMeta(map[string]any{"title": "x", "type": "spec"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Status != models.StatusTodo {
		t.Errorf("default status: got %q, want %q", doc.Status, models.StatusTodo)
	}
}

func TestFromMetaRejectsInvalidStatus(t *testing.T) {
	// The exact values the agent guessed before landing on "in-progress".
	for _, bad := range []string{"active", "wip", "draft", "in_progress"} {
		_, err := FromMeta(map[string]any{"title": "x", "type": "plan", "status": bad}, nil)
		if err == nil {
			t.Fatalf("status %q accepted, want rejection", bad)
		}
		// The error must name the valid set so the caller never has to guess.
		for _, want := range models.ValidStatuses() {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("status %q error %q missing valid value %q", bad, err.Error(), want)
			}
		}
	}
}

func TestFromMetaRejectsInvalidType(t *testing.T) {
	_, err := FromMeta(map[string]any{"title": "x", "type": "task"}, nil)
	if err == nil {
		t.Fatal("type \"task\" accepted, want rejection")
	}
	for _, want := range models.ValidDocTypes() {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("type error %q missing valid value %q", err.Error(), want)
		}
	}
}
