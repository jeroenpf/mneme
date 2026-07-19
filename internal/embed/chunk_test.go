package embed_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/models"
)

func chunkMap(chunks []embed.Chunk) map[string]string {
	m := map[string]string{}
	for _, c := range chunks {
		m[c.ID] = c.Text
	}
	return m
}

func chunkIDs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// The recursive walker emits one chunk per addressable block keyed by its
// stable id, recursing through nested children, and skips id-less and
// empty-text blocks. Chunks are non-overlapping: a section carries only its
// own text, not its children's.
func TestChunksRecursiveWalkerPerBlock(t *testing.T) {
	doc := &models.Document{
		Title:   "Design Doc",
		Project: ptrs("apollo"),
		Body: map[string]any{"sections": []any{
			map[string]any{
				"type": "section", "id": "sec1", "title": "Intro", "content": "the intro prose",
				"children": []any{
					map[string]any{"type": "text", "id": "txt1", "content": "nested note"},
					map[string]any{"type": "section", "id": "sec2", "title": "Details", "content": "detail prose"},
				},
			},
			map[string]any{"type": "callout", "id": "call1", "variant": "info", "title": "Heads up", "content": "callout body"},
			map[string]any{"type": "code", "id": "code1", "lang": "go", "filename": "main.go", "content": "package main"},
			map[string]any{"type": "section", "title": "No ID here", "content": "should be skipped"}, // id-less
			map[string]any{"type": "section", "id": "empty1"},                                        // empty text
		}},
	}
	m := chunkMap(embed.Chunks(doc))

	// One chunk per addressable block; id-less and empty-text blocks skipped.
	for _, id := range []string{"sec1", "txt1", "sec2", "call1", "code1"} {
		if _, ok := m[id]; !ok {
			t.Errorf("missing chunk for block %q; got ids %v", id, chunkIDs(m))
		}
	}
	if len(m) != 5 {
		t.Errorf("expected 5 chunks, got %d: %v", len(m), chunkIDs(m))
	}

	// Non-overlapping: sec1 carries its own prose but not its children's.
	if !strings.Contains(m["sec1"], "the intro prose") {
		t.Errorf("sec1 chunk missing own content: %q", m["sec1"])
	}
	if strings.Contains(m["sec1"], "nested note") || strings.Contains(m["sec1"], "detail prose") {
		t.Errorf("sec1 chunk leaked child content (must be non-overlapping): %q", m["sec1"])
	}
	// Children are their own chunks.
	if !strings.Contains(m["txt1"], "nested note") {
		t.Errorf("txt1 chunk missing content: %q", m["txt1"])
	}
	if !strings.Contains(m["sec2"], "detail prose") {
		t.Errorf("sec2 chunk missing content: %q", m["sec2"])
	}
	// Every chunk carries the document breadcrumb (title + project) for context.
	for id, text := range m {
		if !strings.Contains(text, "Design Doc") || !strings.Contains(text, "apollo") {
			t.Errorf("chunk %q missing breadcrumb: %q", id, text)
		}
	}
}

// A document with no addressable blocks still embeds one `full` chunk on
// title+project.
func TestChunksEmptyDocumentFallback(t *testing.T) {
	doc := &models.Document{Title: "Bare", Project: ptrs("apollo"), Body: map[string]any{}}
	chunks := embed.Chunks(doc)
	if len(chunks) != 1 || chunks[0].ID != "full" {
		t.Fatalf("expected one `full` fallback chunk, got %+v", chunks)
	}
	if !strings.Contains(chunks[0].Text, "Bare") {
		t.Errorf("fallback chunk missing title: %q", chunks[0].Text)
	}
}

func assertContains(t *testing.T, m map[string]string, id string, subs ...string) {
	t.Helper()
	text, ok := m[id]
	if !ok {
		t.Errorf("missing chunk %q; got ids %v", id, chunkIDs(m))
		return
	}
	for _, s := range subs {
		if !strings.Contains(text, s) {
			t.Errorf("chunk %q missing %q in %q", id, s, text)
		}
	}
}

// Every structured block type contributes its salient text, and task entries
// inside subphases and task-lists are indexed as their own chunks keyed by the
// task id (with a done/todo status marker).
func TestChunksIndexesStructuredBlockTypesAndTasks(t *testing.T) {
	doc := &models.Document{
		Title:   "Plan",
		Project: ptrs("apollo"),
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "subphase", "id": "sp1", "num": float64(4), "title": "Retrieval", "description": "make search good",
				"tasks": []any{
					map[string]any{"id": "sp1-t1", "title": "recursive chunker", "content": "walk the AST", "done": true},
					map[string]any{"id": "sp1-t2", "title": "fusion normalize", "done": false},
				},
			},
			map[string]any{"type": "callout", "id": "c1", "variant": "warning", "title": "Careful", "content": "watch out"},
			map[string]any{"type": "code", "id": "cd1", "lang": "sql", "filename": "search.sql", "content": "SELECT 1"},
			map[string]any{"type": "table", "id": "tb1", "title": "Matrix",
				"cols": []any{"Stage", "Focus"},
				"rows": []any{[]any{"1", "Identity"}, []any{"2", "Correctness"}}},
			map[string]any{"type": "key-value", "id": "kv1", "title": "Gates",
				"data": map[string]any{"Retrieval": "every block indexed", "Local": "network minimal"}},
			map[string]any{"type": "task-list", "id": "tl1", "title": "Checklist",
				"tasks": []any{map[string]any{"id": "tl1-t1", "title": "ship it", "content": "make build"}}},
		}},
	}
	m := chunkMap(embed.Chunks(doc))

	assertContains(t, m, "sp1", "4", "Retrieval", "make search good")                                    // subphase: num+title+description
	assertContains(t, m, "sp1-t1", "recursive chunker", "walk the AST", "done")                          // task under subphase
	assertContains(t, m, "sp1-t2", "fusion normalize", "todo")                                           // incomplete task status
	assertContains(t, m, "c1", "warning", "Careful", "watch out")                                        // callout: variant included
	assertContains(t, m, "cd1", "search.sql", "sql", "SELECT 1")                                         // code: filename+lang+content
	assertContains(t, m, "tb1", "Matrix", "Stage", "Focus", "Identity", "Correctness")                   // table cells
	assertContains(t, m, "kv1", "Gates", "Retrieval", "every block indexed", "Local", "network minimal") // key-value pairs
	assertContains(t, m, "tl1-t1", "ship it", "make build")                                              // task-list task

	// Container blocks and their nested entries are distinct chunks.
	if _, ok := m["tl1"]; !ok {
		t.Errorf("task-list container chunk missing; got %v", chunkIDs(m))
	}
}

// key-value data is emitted in a stable (sorted-key) order so unchanged
// content never produces a different chunk text and spurious re-embeds.
func TestChunksKeyValueDeterministicOrder(t *testing.T) {
	mk := func() string {
		doc := &models.Document{Title: "D", Project: ptrs("p"), Body: map[string]any{"sections": []any{
			map[string]any{"type": "key-value", "id": "kv", "data": map[string]any{
				"alpha": "1", "beta": "2", "gamma": "3", "delta": "4"}},
		}}}
		return chunkMap(embed.Chunks(doc))["kv"]
	}
	first := mk()
	for i := 0; i < 25; i++ {
		if got := mk(); got != first {
			t.Fatalf("key-value chunk text not deterministic:\n%q\nvs\n%q", first, got)
		}
	}
}
