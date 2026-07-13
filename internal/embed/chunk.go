package embed

import (
	"fmt"
	"strings"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// Chunk is one unit of embeddable text with a stable id (a section id for
// documents, "full" for the single-chunk entity types).
type Chunk struct {
	ID   string
	Text string
}

// Chunks extracts embeddable chunks from a source model. Documents chunk
// per body section; the short entity types embed as a single chunk.
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
	walkSections(sections, d.Title, deref(d.Project), &out)
	if len(out) == 0 {
		// A document with no sections still embeds as one chunk on title+project.
		out = append(out, Chunk{ID: "full", Text: join(d.Title, deref(d.Project))})
	}
	return out
}

// walkSections recurses body blocks, emitting one chunk per section block.
// Mirrors the recursion in internal/mcp/blocks.go walkBlocks.
func walkSections(blocks []any, title, project string, out *[]Chunk) {
	for _, b := range blocks {
		m, ok := b.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == "section" {
			id, _ := m["id"].(string)
			st, _ := m["title"].(string)
			content, _ := m["content"].(string)
			if id != "" {
				*out = append(*out, Chunk{
					ID:   id,
					Text: fmt.Sprintf("%s | %s | %s: %s", title, project, st, strings.TrimSpace(content)),
				})
			}
		}
		if kids, ok := m["children"].([]any); ok {
			walkSections(kids, title, project, out)
		}
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
