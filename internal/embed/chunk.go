package embed

import (
	"strings"

	"github.com/jeroenpfeil/mneme/internal/models"
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
		if kids, ok := m["children"].([]any); ok {
			walkChunks(kids, title, project, out)
		}
	}
}

// blockText renders a single block's own salient text, excluding nested
// children (which are chunked separately). A generic title+content covers the
// prose block types; the structured types (code, table, key-value, diagram,
// subphase, callout) and task entries get richer extraction in road-p4-t2.
func blockText(typ string, m map[string]any) string {
	title, _ := m["title"].(string)
	content, _ := m["content"].(string)
	return join(strings.TrimSpace(title), strings.TrimSpace(content))
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
