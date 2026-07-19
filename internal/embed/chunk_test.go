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
