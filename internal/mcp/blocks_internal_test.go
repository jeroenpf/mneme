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

func TestDetectBlockMarkdown(t *testing.T) {
	flagged := map[string]string{
		"one\n\ntwo":  "multiple paragraphs",
		"- a\n- b":    "a list",
		"1. a\n2. b":  "a list",
		"# Heading":   "a heading",
		"```\nx\n```": "a fenced code block",
	}
	for in, want := range flagged {
		if got := detectBlockMarkdown(in); got != want {
			t.Errorf("detectBlockMarkdown(%q) = %q, want %q", in, got, want)
		}
	}
	for _, safe := range []string{
		"Inline **bold**, `code`, and a mid-line - dash.",
		"Version 1.2 and item 3 stay inline.",
		"A single\nnewline is fine.",
	} {
		if got := detectBlockMarkdown(safe); got != "" {
			t.Errorf("detectBlockMarkdown(%q) = %q, want none", safe, got)
		}
	}
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

func TestValidateBodyRejectsBlockMarkdownInProse(t *testing.T) {
	err := validateBody(sectionBody(
		map[string]any{"type": "text", "id": "p", "content": "**What:** a\n\n**Why:** b"},
	))
	if err == nil {
		t.Fatal("expected rejection of paragraph break in text.content")
	}
	if !strings.Contains(err.Error(), "inline-only") ||
		!strings.Contains(err.Error(), "child blocks") {
		t.Errorf("error must teach the fix, got %q", err.Error())
	}
}

func TestValidateBodyAllowsNewlinesInCodeAndDiagram(t *testing.T) {
	if err := validateBody(sectionBody(
		map[string]any{"type": "code", "id": "c", "lang": "go", "content": "a\n\nb"},
		map[string]any{"type": "diagram", "id": "d", "content": "flowchart LR\n\n A-->B"},
	)); err != nil {
		t.Fatalf("code/diagram newlines must be allowed: %v", err)
	}
}

func TestValidateBodyRejectsBlockMarkdownInCollections(t *testing.T) {
	// task-list: a task whose content field is a bullet list.
	err := validateBody(sectionBody(
		map[string]any{"type": "task-list", "id": "tl", "tasks": []any{
			map[string]any{"id": "x", "title": "ok", "content": "- one\n- two"},
		}},
	))
	if err == nil {
		t.Fatal("expected rejection of a list in a task-list task's content")
	}
	if !strings.Contains(err.Error(), "inline-only") {
		t.Errorf("task-list error must teach inline-only, got %q", err.Error())
	}

	// key-value: a value that is a bullet list.
	err = validateBody(sectionBody(
		map[string]any{"type": "key-value", "id": "kv", "data": map[string]any{
			"k": "- one\n- two",
		}},
	))
	if err == nil {
		t.Fatal("expected rejection of a list in a key-value value")
	}
	if !strings.Contains(err.Error(), "inline-only") {
		t.Errorf("key-value error must teach inline-only, got %q", err.Error())
	}
}
