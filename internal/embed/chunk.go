package embed

import (
	"sort"
	"strconv"
	"strings"

	"github.com/jeroenpf/mneme/internal/models"
)

// Chunk is one unit of embeddable text with a stable id — a block/task id for
// documents, "full" for the single-chunk entity types.
type Chunk struct {
	ID   string
	Text string
}

// Chunks extracts embeddable chunks from a source model. Documents chunk per
// addressable body block (recursively); the short entity types embed as a
// single chunk.
func Chunks(src any) []Chunk {
	switch s := src.(type) {
	case *models.Document:
		return documentChunks(s)
	case *models.Decision:
		return []Chunk{{ID: "full", Text: join(s.Title, deref(s.Project), s.Decision, s.Rationale, s.Alternatives, s.Consequences)}}
	case *models.Snippet:
		return []Chunk{{ID: "full", Text: join(s.Title, deref(s.Project), s.Description, s.Content)}}
	case *models.Solution:
		return []Chunk{{ID: "full", Text: join(s.ErrorDescription, deref(s.Project), s.Solution)}}
	case *models.JournalEntry:
		return []Chunk{{ID: "full", Text: join(s.Summary, deref(s.Project), s.SessionRef,
			strings.Join(s.Accomplished, " "), strings.Join(s.Deferred, " "))}}
	default:
		return nil
	}
}

func documentChunks(d *models.Document) []Chunk {
	sections, _ := d.Body["sections"].([]any)
	var out []Chunk
	walkChunks(sections, d.Title, deref(d.Project), &out)
	if len(out) == 0 {
		// A document with no addressable blocks still embeds one chunk on
		// title+project.
		out = append(out, Chunk{ID: "full", Text: join(d.Title, deref(d.Project))})
	}
	return out
}

// walkChunks recurses body blocks, emitting one chunk per addressable block
// keyed by its stable id. Non-overlapping: a block's chunk carries only its
// own text; nested children are chunked separately. Blocks with no id or no
// extractable text are skipped.
func walkChunks(blocks []any, title, project string, out *[]Chunk) {
	for _, b := range blocks {
		m, ok := b.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		typ, _ := m["type"].(string)
		if id != "" {
			if text := blockText(typ, m); text != "" {
				*out = append(*out, Chunk{ID: id, Text: join(title, project, text)})
			}
		}
		// Task entries (subphase.tasks, task-list.tasks) are their own chunks
		// keyed by task id — the addressable unit for "what's next" retrieval.
		if tasks, ok := m["tasks"].([]any); ok {
			for _, tr := range tasks {
				tm, ok := tr.(map[string]any)
				if !ok {
					continue
				}
				tid, _ := tm["id"].(string)
				if tid == "" {
					continue
				}
				if text := taskText(tm); text != "" {
					*out = append(*out, Chunk{ID: tid, Text: join(title, project, text)})
				}
			}
		}
		if kids, ok := m["children"].([]any); ok {
			walkChunks(kids, title, project, out)
		}
	}
}

// blockText renders a single block's own salient text, excluding nested
// children (which are chunked separately). Structured types flatten their
// distinctive fields; the prose types fall back to title+content.
func blockText(typ string, m map[string]any) string {
	switch typ {
	case "subphase":
		return join(numStr(m["num"]), str(m, "title"), str(m, "description"))
	case "callout":
		return join(str(m, "variant"), str(m, "title"), str(m, "content"))
	case "code":
		return join(str(m, "filename"), str(m, "lang"), str(m, "content"))
	case "table":
		return join(str(m, "title"), tableText(m))
	case "key-value":
		return join(str(m, "title"), keyValueText(m))
	default:
		// section, text, diagram, task-list, and any other prose block.
		return join(str(m, "title"), str(m, "content"))
	}
}

// taskText renders one task entry: its title, content, and a done/todo status
// marker so status-aware queries can reach it. Empty tasks are skipped.
func taskText(tm map[string]any) string {
	base := join(str(tm, "title"), str(tm, "content"))
	if base == "" {
		return ""
	}
	status := "todo"
	if done, _ := tm["done"].(bool); done {
		status = "done"
	}
	return join(base, status)
}

// tableText flattens a table's header cells then row cells in document order.
func tableText(m map[string]any) string {
	var parts []string
	if cols, ok := m["cols"].([]any); ok {
		for _, c := range cols {
			if s, ok := c.(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}
	}
	if rows, ok := m["rows"].([]any); ok {
		for _, r := range rows {
			cells, ok := r.([]any)
			if !ok {
				continue
			}
			for _, cell := range cells {
				if s, ok := cell.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
		}
	}
	return join(parts...)
}

// keyValueText flattens a key-value block's data map into "key: value" pairs
// in sorted-key order so identical content always yields identical chunk text
// (Go map iteration is otherwise random, which would churn embeddings).
func keyValueText(m map[string]any) string {
	data, ok := m["data"].(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v, _ := data[k].(string)
		parts = append(parts, k+": "+strings.TrimSpace(v))
	}
	return join(parts...)
}

// str reads a trimmed string field, tolerating a missing or non-string value.
func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return strings.TrimSpace(s)
}

// numStr stringifies a JSON number (float64) as a plain integer when it has no
// fractional part, leaving strings intact and anything else empty.
func numStr(v any) string {
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n)
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'g', -1, 64)
	default:
		return ""
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// join concatenates non-empty parts with " | " so the embedding sees the
// salient fields the way each type's FTS vector weights them.
func join(parts ...string) string {
	kept := parts[:0]
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " | ")
}
