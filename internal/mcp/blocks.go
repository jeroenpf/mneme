package mcp

import (
	"fmt"
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
