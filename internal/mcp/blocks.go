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
	if err := walkBlocks(arr, "body.sections"); err != nil {
		return err
	}
	return validatePlanStructure(body)
}

// validatePlanStructure enforces plan invariants: subphases form a flat
// sequence (never nested inside another subphase) with unique, strictly
// increasing nums in document order. Existing documents only re-validate
// on their next full push, so increasing-not-consecutive keeps every
// stored plan upsertable.
func validatePlanStructure(body map[string]any) error {
	var nums []int
	var walk func(blocks []any, inSubphase bool, path string) error
	walk = func(blocks []any, inSubphase bool, path string) error {
		for i, raw := range blocks {
			b, ok := raw.(map[string]any)
			if !ok {
				continue // walkBlocks already rejected non-objects
			}
			p := fmt.Sprintf("%s[%d]", path, i)
			typ, _ := b["type"].(string)
			if typ == "subphase" {
				if inSubphase {
					return fmt.Errorf("%s: a subphase cannot nest inside another subphase — phases are a flat sequence; use a section or task-list inside the phase", p)
				}
				if n, ok := b["num"].(float64); ok {
					nums = append(nums, int(n))
				}
			}
			if children, ok := b["children"].([]any); ok {
				if err := walk(children, inSubphase || typ == "subphase", p+".children"); err != nil {
					return err
				}
			}
		}
		return nil
	}
	sections, _ := body["sections"].([]any)
	if err := walk(sections, false, "body.sections"); err != nil {
		return err
	}
	for i := 1; i < len(nums); i++ {
		if nums[i] == nums[i-1] {
			return fmt.Errorf("subphase num %d appears twice — nums must be unique", nums[i])
		}
		if nums[i] < nums[i-1] {
			return fmt.Errorf("subphase nums must increase in document order; %d follows %d", nums[i], nums[i-1])
		}
	}
	return nil
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

// countSubphases reports how many subphase blocks the body holds,
// descending into children so a subphase wrapped in a section is still
// counted. Used to keep meta.phase_total consistent with the body.
func countSubphases(body map[string]any) int {
	var count func(blocks []any) int
	count = func(blocks []any) int {
		n := 0
		for _, raw := range blocks {
			b, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "subphase" {
				n++
			}
			if children, ok := b["children"].([]any); ok {
				n += count(children)
			}
		}
		return n
	}
	sections, _ := body["sections"].([]any)
	return count(sections)
}

// --- inline-only content validation ----------------------------------

var (
	reList    = regexp.MustCompile(`(?m)^[ \t]*[-*+] `)
	reOrdered = regexp.MustCompile(`(?m)^[ \t]*\d+[.)] `)
	reHeading = regexp.MustCompile(`(?m)^#{1,6} `)
	reFence   = regexp.MustCompile("(?m)^```")
)

// detectBlockStructure returns a human label for the first list, heading,
// or fenced-code construct in s — block markdown that maps to a distinct
// typed block and cannot render inside a prose field — or "" when none.
// A string with no newline is never a list: a lone leading "1. " or "- "
// renders literally in both inline and paragraph modes, so flagging it is
// a false positive (a numbered title, not a list).
func detectBlockStructure(s string) string {
	switch {
	case strings.Contains(s, "\n") && (reList.MatchString(s) || reOrdered.MatchString(s)):
		return "a list"
	case reHeading.MatchString(s):
		return "a heading"
	case reFence.MatchString(s):
		return "a fenced code block"
	}
	return ""
}

// detectBlockMarkdown extends detectBlockStructure with the paragraph-break
// check, for strictly-inline fields (titles, task titles, key-value values)
// that render as a single line where even a blank line would collapse.
func detectBlockMarkdown(s string) string {
	if sig := detectBlockStructure(s); sig != "" {
		return sig
	}
	if strings.Contains(s, "\n\n") {
		return "multiple paragraphs"
	}
	return ""
}

// paragraphProseFields render through renderParagraphs on the client: body
// prose that carries paragraph flow. Blank-line paragraphs are allowed —
// a paragraph break is prose, not structure — but a list, heading, or code
// fence still maps to a distinct typed block and is rejected here.
var paragraphProseFields = map[string][]string{
	"section":  {"content"},
	"text":     {"content"},
	"callout":  {"content"},
	"subphase": {"description"},
}

// inlineProseFields render strictly inline through renderInline — titles and
// other single-line fields where even a paragraph break flattens. Every
// block-level construct, blank lines included, is rejected. Fields absent
// from both maps are exempt on purpose: code.content and diagram.content
// carry newline-significant source.
var inlineProseFields = map[string][]string{
	"section":  {"title"},
	"callout":  {"title"},
	"subphase": {"title"},
}

// validateInlineFields enforces the prose contract per block type: body
// prose accepts paragraphs but not list/heading/fence structure; titles and
// collection values (task titles, key-value values) accept neither. Both
// teach the caller to move real structure into typed child blocks.
func validateInlineFields(b map[string]any, typ, path string) error {
	for _, f := range paragraphProseFields[typ] {
		s, _ := b[f].(string)
		if sig := detectBlockStructure(s); sig != "" {
			return fmt.Errorf("%s: %s.%s contains %s, but this field renders "+
				"paragraphs of inline markdown only. Move it into a typed child block: "+
				"a {type:\"task-list\",...} for a list, {type:\"code\",...} for a code "+
				"block, or a nested {type:\"section\",...} for a heading. Plain "+
				"paragraphs are fine — separate them with a blank line. "+
				"See push_document for block shapes.", path, typ, f, sig)
		}
	}
	for _, f := range inlineProseFields[typ] {
		s, _ := b[f].(string)
		if sig := detectBlockMarkdown(s); sig != "" {
			return fmt.Errorf("%s: %s.%s contains %s, but this field renders "+
				"inline-only — it must stay a single line. Move any structure into "+
				"child blocks. See push_document for block shapes.", path, typ, f, sig)
		}
	}
	// tasks and data are collections, not walked as children, so scan here.
	switch typ {
	case "task-list", "subphase":
		tasks, _ := b["tasks"].([]any)
		for i, raw := range tasks {
			task, _ := raw.(map[string]any)
			if err := validateTaskInline(task, fmt.Sprintf("%s.tasks[%d]", path, i)); err != nil {
				return err
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

// validateTaskInline rejects block markdown in a task's inline fields —
// shared by body validation (task-list and subphase tasks) and the task
// write tools (add_task, update_task).
func validateTaskInline(task map[string]any, path string) error {
	for _, f := range []string{"title", "content"} {
		s, _ := task[f].(string)
		if sig := detectBlockMarkdown(s); sig != "" {
			return fmt.Errorf("%s.%s contains %s; task fields render inline-only — "+
				"keep each task terse, or move detail into a text/callout child block.", path, f, sig)
		}
	}
	return nil
}
