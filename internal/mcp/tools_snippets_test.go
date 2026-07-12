package mcp_test

import "testing"

func TestSaveSnippetCreateAndGet(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Language string `json:"language"`
	}
	call(t, cs, "save_snippet", map[string]any{
		"title":    "Cursor pagination",
		"project":  "apollo",
		"language": "TypeScript",
		"content":  "keyset on id",
		"tags":     []string{"pagination"},
	}, &created)
	if created.ID == "" {
		t.Fatal("expected a generated id")
	}
	if created.Language != "typescript" { // lowercased at the boundary
		t.Errorf("language: got %q, want typescript", created.Language)
	}

	var listed struct {
		Snippets []struct {
			Title string `json:"title"`
		} `json:"snippets"`
	}
	call(t, cs, "get_snippets", map[string]any{"project": "apollo"}, &listed)
	if len(listed.Snippets) != 1 || listed.Snippets[0].Title != "Cursor pagination" {
		t.Fatalf("get_snippets: %+v", listed.Snippets)
	}
}

func TestSaveSnippetUnknownProject(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "save_snippet", map[string]any{
		"title": "x", "project": "ghost", "content": "c",
	})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestSaveSnippetMissingTitle(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "save_snippet", map[string]any{"content": "c"})
	if msg == "" {
		t.Error("expected missing-title error")
	}
}

func TestSaveSnippetMissingContent(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "save_snippet", map[string]any{"title": "t"})
	if msg == "" {
		t.Error("expected missing-content error")
	}
}

func TestSaveSnippetUpsert(t *testing.T) {
	cs := newClient(t)
	var created struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	call(t, cs, "save_snippet", map[string]any{
		"title": "HTTP retry", "content": "old", "tags": []string{"net"},
	}, &created)

	var updated struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	call(t, cs, "save_snippet", map[string]any{
		"id": created.ID, "content": "new body", "tags": []string{},
	}, &updated)
	if updated.Content != "new body" {
		t.Errorf("content not updated: %q", updated.Content)
	}
	if updated.Title != "HTTP retry" { // partial update preserves title
		t.Errorf("title clobbered: %q", updated.Title)
	}
	if len(updated.Tags) != 0 { // explicit [] clears tags
		t.Errorf("tags not cleared: %+v", updated.Tags)
	}
}

func TestSearchSnippetsTool(t *testing.T) {
	cs := newClient(t)
	call(t, cs, "save_snippet", map[string]any{
		"title": "Cursor pagination", "content": "keyset on id",
		"description": "Stable pages under concurrent inserts.",
	}, nil)
	call(t, cs, "save_snippet", map[string]any{
		"title": "Errgroup fan-out", "content": "errgroup.WithContext",
		"description": "Bounded concurrency for parallel calls.",
	}, nil)

	var out struct {
		Snippets []struct {
			Title string `json:"title"`
		} `json:"snippets"`
	}
	call(t, cs, "search_snippets", map[string]any{"query": "concurrent pages"}, &out)
	if len(out.Snippets) == 0 || out.Snippets[0].Title != "Cursor pagination" {
		t.Fatalf("search_snippets ranking: %+v", out.Snippets)
	}
}
