package mcp

import (
	"strings"
	"testing"
)

// section wraps blocks in the body shape validateBody expects.
func sectionBody(children ...any) map[string]any {
	return map[string]any{"sections": []any{
		map[string]any{"type": "section", "id": "s", "title": "S", "children": children},
	}}
}

func TestValidateBodyAcceptsCanonicalBlocks(t *testing.T) {
	body := sectionBody(
		map[string]any{"type": "text", "id": "p", "content": "prose"},
		map[string]any{"type": "code", "id": "c", "lang": "go", "filename": "main.go", "content": "package main"},
		map[string]any{"type": "diagram", "id": "d", "content": "flowchart LR\n A-->B"},
		map[string]any{"type": "table", "id": "tb", "cols": []any{"a"}, "rows": []any{[]any{"1"}}},
		map[string]any{"type": "callout", "id": "ca", "variant": "info", "content": "hi"},
		map[string]any{"type": "key-value", "id": "kv", "data": map[string]any{"k": "v"}},
	)
	if err := validateBody(body); err != nil {
		t.Fatalf("canonical blocks rejected: %v", err)
	}
}

func TestValidateBodyAcceptsSectionContent(t *testing.T) {
	// The migrated-plan shape: prose on section.content, no children.
	body := map[string]any{"sections": []any{
		map[string]any{"type": "section", "id": "sum", "title": "Summary", "content": "text"},
	}}
	if err := validateBody(body); err != nil {
		t.Fatalf("section.content rejected: %v", err)
	}
}

func TestValidateBodyRejectsMisnamedFields(t *testing.T) {
	cases := []struct {
		name    string
		block   map[string]any
		suggest string // the field name the error should point at
	}{
		{"code.language", map[string]any{"type": "code", "id": "c", "language": "go", "content": "x"}, "lang"},
		{"table.headers", map[string]any{"type": "table", "id": "t", "headers": []any{"a"}, "rows": []any{}}, "cols"},
		{"diagram.mermaid", map[string]any{"type": "diagram", "id": "d", "mermaid": "flowchart"}, "content"},
		{"key-value.pairs", map[string]any{"type": "key-value", "id": "k", "pairs": map[string]any{}}, "data"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateBody(sectionBody(c.block))
			if err == nil {
				t.Fatalf("expected rejection of %s", c.name)
			}
			if !strings.Contains(err.Error(), c.suggest) {
				t.Errorf("error %q should point at %q", err.Error(), c.suggest)
			}
		})
	}
}

func TestValidateBodyRejectsUnknownFieldWithAllowedList(t *testing.T) {
	err := validateBody(sectionBody(
		map[string]any{"type": "code", "id": "c", "content": "x", "theme": "dracula"},
	))
	if err == nil {
		t.Fatal("expected rejection of unknown field 'theme'")
	}
	// The error must enumerate the allowed fields so the caller never guesses.
	for _, want := range []string{"lang", "filename", "content"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing allowed field %q", err.Error(), want)
		}
	}
}
