package mcp

import (
	"reflect"
	"strings"
	"testing"
)

// lintBody is the collect-mode twin of validateBody: one hit per violation,
// never fail-fast, covering structural problems and every inline rule.
func TestLintBodyCollectsEveryViolationClass(t *testing.T) {
	body := sectionBody(
		map[string]any{"type": "widget", "id": "w"},
		map[string]any{"type": "code", "id": "c", "content": "x", "theme": "dracula"},
		map[string]any{"type": "text", "id": "p", "content": "- a\n- b"},
		map[string]any{"type": "callout", "id": "ca", "variant": "note", "title": "one\n\ntwo", "content": "ok"},
		map[string]any{"type": "subphase", "id": "sp", "num": "1", "title": "P", "tasks": []any{
			map[string]any{"id": "t", "title": "ok", "content": "1. a\n2. b"},
		}},
		map[string]any{"type": "key-value", "id": "kv", "data": map[string]any{"k": "- one\n- two"}},
	)
	want := []lintHit{
		{BlockID: "w", Path: "body.sections[0].children[0]", Field: "type", Found: `unknown block type "widget"`},
		{BlockID: "c", Path: "body.sections[0].children[1]", Field: "theme", Found: "unknown field"},
		{BlockID: "p", Path: "body.sections[0].children[2]", Field: "content", Found: "a list", Excerpt: "- a ⏎ - b"},
		{BlockID: "ca", Path: "body.sections[0].children[3]", Field: "title", Found: "multiple paragraphs", Excerpt: "one ⏎  ⏎ two"},
		{BlockID: "t", Path: "body.sections[0].children[4].tasks[0]", Field: "content", Found: "a list", Excerpt: "1. a ⏎ 2. b"},
		{BlockID: "kv", Path: "body.sections[0].children[5]", Field: `data["k"]`, Found: "a list", Excerpt: "- one ⏎ - two"},
	}
	got := lintBody(body)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lintBody hits mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestLintBodyCleanAndNil(t *testing.T) {
	if got := lintBody(nil); got != nil {
		t.Errorf("lintBody(nil) = %+v, want nil", got)
	}
	clean := sectionBody(
		map[string]any{"type": "text", "id": "p", "content": "plain prose\n\ntwo paragraphs"},
		map[string]any{"type": "code", "id": "c", "lang": "go", "content": "- not\n- linted"},
	)
	if got := lintBody(clean); len(got) != 0 {
		t.Errorf("clean body produced hits: %+v", got)
	}
}

func TestExcerptFlattensAndCaps(t *testing.T) {
	if got := excerpt("a\nb"); got != "a ⏎ b" {
		t.Errorf("excerpt flatten = %q", got)
	}
	long := strings.Repeat("x", 100)
	got := excerpt(long)
	if want := strings.Repeat("x", 80) + "…"; got != want {
		t.Errorf("excerpt cap = %q (len %d)", got, len([]rune(got)))
	}
}
