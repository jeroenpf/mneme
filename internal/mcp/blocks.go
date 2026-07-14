package mcp

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// validBlockTypes is the closed set of block.type values accepted by
// push_document and the section/task editing tools. It aligns with the
// Vue renderer components added in phase 1.5.
var validBlockTypes = map[string]bool{
	"section":   true,
	"subphase":  true,
	"task-list": true,
	"callout":   true,
	"code":      true,
	"table":     true,
	"diagram":   true,
	"key-value": true,
	"text":      true,
}

// blockFields lists the non-universal field keys each block type accepts.
// `id` and `type` are accepted on every block. These names match the Vue
// renderer components exactly: a block that renders nothing is almost
// always a misnamed field (language vs lang, headers vs cols), so writes
// reject unknown fields loudly rather than silently dropping them — the
// same silent-drop trap that hid section prose.
var blockFields = map[string][]string{
	"section":   {"title", "content", "children"},
	"subphase":  {"num", "title", "session", "description", "tasks", "children"},
	"text":      {"content"},
	"task-list": {"title", "tasks"},
	"callout":   {"variant", "title", "content"},
	"code":      {"lang", "filename", "content"},
	"table":     {"title", "cols", "rows"},
	"diagram":   {"title", "content"},
	"key-value": {"title", "data"},
}

// fieldAliases maps common wrong field names to the correct one so the
// error can say "did you mean X?" for the usual mistakes.
var fieldAliases = map[string]string{
	"language": "lang",
	"headers":  "cols",
	"header":   "cols",
	"columns":  "cols",
	"mermaid":  "content",
	"source":   "content",
	"code":     "content",
	"text":     "content",
	"body":     "content",
	"prose":    "content",
	"pairs":    "data",
	"items":    "data",
	"kv":       "data",
}

// validateBody walks body.sections recursively and returns an error on
// the first block with an unknown type or an unknown/misnamed field. A
// section's children are walked too; task entries carry no type field and
// are not validated for shape here.
func validateBody(body map[string]any) error {
	if body == nil {
		return nil
	}
	sections, ok := body["sections"]
	if !ok {
		return nil
	}
	arr, ok := sections.([]any)
	if !ok {
		return fmt.Errorf("body.sections must be an array")
	}
	return walkBlocks(arr, "body.sections")
}

func walkBlocks(blocks []any, path string) error {
	for i, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", path, i)
		}
		t, _ := b["type"].(string)
		if t == "" {
			return fmt.Errorf("%s[%d] missing 'type'", path, i)
		}
		if !validBlockTypes[t] {
			return fmt.Errorf("%s[%d] has invalid type %q", path, i, t)
		}
		if err := validateBlockFields(b, t, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
		if err := validateInlineFields(b, t, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
		if children, ok := b["children"].([]any); ok {
			if err := walkBlocks(children, fmt.Sprintf("%s[%d].children", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateBlockFields rejects keys not in the block type's allowed set,
// pointing at the likely intended field for common misnamings.
func validateBlockFields(b map[string]any, t, path string) error {
	allowed := blockFields[t]
	for k := range b {
		if k == "id" || k == "type" || slices.Contains(allowed, k) {
			continue
		}
		hint := ""
		if canon, ok := fieldAliases[k]; ok && slices.Contains(allowed, canon) {
			hint = fmt.Sprintf(" (did you mean %q?)", canon)
		}
		return fmt.Errorf("%s: %s block has unknown field %q%s; allowed fields: id, type, %s",
			path, t, k, hint, strings.Join(allowed, ", "))
	}
	return nil
}

// --- inline-only content validation ----------------------------------

var (
	reList    = regexp.MustCompile(`(?m)^[ \t]*[-*+] `)
	reOrdered = regexp.MustCompile(`(?m)^[ \t]*\d+[.)] `)
	reHeading = regexp.MustCompile(`(?m)^#{1,6} `)
	reFence   = regexp.MustCompile("(?m)^```")
)

// detectBlockMarkdown returns a human label for the first block-level
// markdown construct in s, or "" when s is safe for inline rendering.
// These are exactly the constructs renderInline (inline-only) flattens.
func detectBlockMarkdown(s string) string {
	switch {
	case reList.MatchString(s) || reOrdered.MatchString(s):
		return "a list"
	case reHeading.MatchString(s):
		return "a heading"
	case reFence.MatchString(s):
		return "a fenced code block"
	case strings.Contains(s, "\n\n"):
		return "multiple paragraphs"
	}
	return ""
}

// inlineProseFields lists the string fields per block type that render
// through renderInline. Fields absent here are exempt on purpose:
// code.content and diagram.content carry newline-significant source.
var inlineProseFields = map[string][]string{
	"section":  {"title", "content"},
	"text":     {"content"},
	"callout":  {"title", "content"},
	"subphase": {"title", "description"},
}

// validateInlineFields rejects block-level markdown in a block's
// inline-only prose fields, teaching the caller to split into children.
func validateInlineFields(b map[string]any, typ, path string) error {
	for _, f := range inlineProseFields[typ] {
		s, _ := b[f].(string)
		if sig := detectBlockMarkdown(s); sig != "" {
			return fmt.Errorf("%s: %s.%s contains %s, but this field renders "+
				"inline-only — newlines, lists, and headings collapse to one line. "+
				"Split structure into child blocks: one {type:\"text\",content:...} "+
				"per paragraph, a {type:\"callout\",...} for notes, or {type:\"code\",...}. "+
				"See push_document for block shapes.", path, typ, f, sig)
		}
	}
	// tasks and data are collections, not walked as children, so scan here.
	switch typ {
	case "task-list":
		tasks, _ := b["tasks"].([]any)
		for i, raw := range tasks {
			task, _ := raw.(map[string]any)
			for _, f := range []string{"title", "content"} {
				s, _ := task[f].(string)
				if sig := detectBlockMarkdown(s); sig != "" {
					return fmt.Errorf("%s.tasks[%d].%s contains %s; task fields render "+
						"inline-only — keep each task terse, or move detail into a "+
						"text/callout child block.", path, i, f, sig)
				}
			}
		}
	case "key-value":
		data, _ := b["data"].(map[string]any)
		for k, raw := range data {
			s, _ := raw.(string)
			if sig := detectBlockMarkdown(s); sig != "" {
				return fmt.Errorf("%s.data[%q] contains %s; key-value values render "+
					"inline-only. Use short values or a text block for prose.", path, k, sig)
			}
		}
	}
	return nil
}
