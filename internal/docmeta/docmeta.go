// Package docmeta promotes known top-level meta keys onto typed
// Document columns. Both the REST API and the MCP server reuse it so a
// meta object is interpreted the same way regardless of transport.
package docmeta

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// FromMeta builds a Document by promoting known top-level meta keys
// onto typed columns. Unknown keys remain on doc.Meta. Returns an error
// when a known key has the wrong type.
func FromMeta(meta, body map[string]any) (*models.Document, error) {
	d := &models.Document{
		Status: models.StatusTodo,
		Tags:   []string{},
		Meta:   map[string]any{},
		Body:   body,
	}

	for k, v := range meta {
		switch k {
		case "title":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.title must be a string")
			}
			d.Title = s
		case "type":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.type must be a string")
			}
			if !models.IsValidType(s) {
				return nil, fmt.Errorf("meta.type %q is invalid; must be one of: %s", s, strings.Join(models.ValidDocTypes(), ", "))
			}
			d.Type = s
		case "project":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.project must be a string")
			}
			d.Project = &s
		case "category":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.category must be a string")
			}
			d.Category = &s
		case "status":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.status must be a string")
			}
			if !models.IsValidStatus(s) {
				return nil, fmt.Errorf("meta.status %q is invalid; must be one of: %s", s, strings.Join(models.ValidStatuses(), ", "))
			}
			d.Status = s
		case "ticket":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.ticket must be a string")
			}
			d.Ticket = &s
		case "repo":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.repo must be a string")
			}
			d.Repo = &s
		case "tags":
			tags, err := CastStringSlice(v)
			if err != nil {
				return nil, fmt.Errorf("meta.tags: %w", err)
			}
			d.Tags = tags
		case "phase_current":
			n, err := CastInt(v)
			if err != nil {
				return nil, fmt.Errorf("meta.phase_current: %w", err)
			}
			d.PhaseCurrent = &n
			d.Meta[k] = v
		case "phase_total":
			n, err := CastInt(v)
			if err != nil {
				return nil, fmt.Errorf("meta.phase_total: %w", err)
			}
			d.PhaseTotal = &n
			d.Meta[k] = v
		default:
			d.Meta[k] = v
		}
	}
	return d, nil
}

// ApplyTo copies typed columns + Meta from src onto dst, leaving Body,
// ID, and timestamps untouched. Used when patching meta.
func ApplyTo(dst, src *models.Document) {
	dst.Title = src.Title
	dst.Project = src.Project
	dst.Category = src.Category
	dst.Type = src.Type
	dst.Status = src.Status
	dst.Ticket = src.Ticket
	dst.Repo = src.Repo
	dst.Tags = src.Tags
	dst.PhaseCurrent = src.PhaseCurrent
	dst.PhaseTotal = src.PhaseTotal
	dst.Meta = src.Meta
}

func CastStringSlice(v any) ([]string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, errors.New("must be an array of strings")
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, errors.New("must be an array of strings")
		}
		out = append(out, s)
	}
	return out, nil
}

func CastInt(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	}
	return 0, errors.New("must be an integer")
}
