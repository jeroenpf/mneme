package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// sectionOutput is returned from section-editing tools.
type sectionOutput struct {
	Section map[string]any   `json:"section"`
	Doc     *models.Document `json:"doc"`
}

// --- update_section ---------------------------------------------------

type updateSectionInput struct {
	DocID     string         `json:"doc_id" jsonschema:"document id"`
	SectionID string         `json:"section_id" jsonschema:"id of the section block to patch (anywhere in the body tree)"`
	Patch     map[string]any `json:"patch" jsonschema:"fields to set on the section; id and children are protected"`
}

var sectionProtectedFields = map[string]bool{
	"id":       true,
	"children": true,
}

func (t *tools) updateSection(ctx context.Context, _ *sdk.CallToolRequest, in updateSectionInput) (*sdk.CallToolResult, *sectionOutput, error) {
	if in.SectionID == "" {
		return nil, nil, errors.New("section_id is required")
	}
	if len(in.Patch) == 0 {
		return nil, nil, errors.New("patch must contain at least one field")
	}
	for k := range in.Patch {
		if sectionProtectedFields[k] {
			return nil, nil, fmt.Errorf("patch may not modify %q — use add_section/remove_section instead", k)
		}
	}
	if newType, ok := in.Patch["type"].(string); ok && !validBlockTypes[newType] {
		return nil, nil, fmt.Errorf("invalid type %q", newType)
	}

	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}
	_, _, block := walkSectionsByID(sections, in.SectionID)
	if block == nil {
		return nil, nil, fmt.Errorf("section %q not found", in.SectionID)
	}
	for k, v := range in.Patch {
		block[k] = v
	}

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	return nil, &sectionOutput{Section: block, Doc: doc}, nil
}

// --- add_section ------------------------------------------------------

type addSectionInput struct {
	DocID          string         `json:"doc_id" jsonschema:"document id"`
	Section        map[string]any `json:"section" jsonschema:"section block — must include id and type"`
	AfterSectionID string         `json:"after_section_id,omitempty" jsonschema:"insert immediately after this top-level section (otherwise appends)"`
}

func (t *tools) addSection(ctx context.Context, _ *sdk.CallToolRequest, in addSectionInput) (*sdk.CallToolResult, *sectionOutput, error) {
	if in.Section == nil {
		return nil, nil, errors.New("section is required")
	}
	id, _ := in.Section["id"].(string)
	if id == "" {
		return nil, nil, errors.New("section.id is required")
	}
	typ, _ := in.Section["type"].(string)
	if typ == "" {
		return nil, nil, errors.New("section.type is required")
	}
	if !validBlockTypes[typ] {
		return nil, nil, fmt.Errorf("invalid type %q", typ)
	}
	// If the new block has children, validate the whole subtree.
	if err := walkBlocks([]any{in.Section}, "section"); err != nil {
		return nil, nil, err
	}

	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	if doc.Body == nil {
		doc.Body = map[string]any{}
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}

	insertAt := len(sections)
	if in.AfterSectionID != "" {
		found := false
		for i, raw := range sections {
			b, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if bid, _ := b["id"].(string); bid == in.AfterSectionID {
				insertAt = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("after_section_id %q not found at top level", in.AfterSectionID)
		}
	}
	sections = append(sections, nil)
	copy(sections[insertAt+1:], sections[insertAt:])
	sections[insertAt] = in.Section

	setSections(doc.Body, sections)
	if err := t.saveDoc(ctx, doc); err != nil {
		return nil, nil, err
	}
	return nil, &sectionOutput{Section: in.Section, Doc: doc}, nil
}

// --- remove_section ---------------------------------------------------

type removeSectionInput struct {
	DocID     string `json:"doc_id" jsonschema:"document id"`
	SectionID string `json:"section_id" jsonschema:"id of the section block to remove (anywhere in the body tree)"`
}

func (t *tools) removeSection(ctx context.Context, _ *sdk.CallToolRequest, in removeSectionInput) (*sdk.CallToolResult, *okResult, error) {
	if in.SectionID == "" {
		return nil, nil, errors.New("section_id is required")
	}
	doc, err := t.loadDoc(ctx, in.DocID)
	if err != nil {
		return nil, nil, err
	}
	sections, err := sectionsArray(doc.Body)
	if err != nil {
		return nil, nil, err
	}

	if removed := removeBlockByID(sections, in.SectionID); removed {
		setSections(doc.Body, sections)
		if err := t.saveDoc(ctx, doc); err != nil {
			return nil, nil, err
		}
		return nil, &okResult{OK: true}, nil
	}
	// Top-level removal needs to mutate the slice itself (length changes).
	for i, raw := range sections {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if bid, _ := b["id"].(string); bid == in.SectionID {
			sections = append(sections[:i], sections[i+1:]...)
			setSections(doc.Body, sections)
			if err := t.saveDoc(ctx, doc); err != nil {
				return nil, nil, err
			}
			return nil, &okResult{OK: true}, nil
		}
	}
	return nil, nil, fmt.Errorf("section %q not found", in.SectionID)
}

// removeBlockByID descends into "children" arrays and removes the
// first block with the given id. Returns true if removed. Only handles
// nested removals — top-level removal must be done by the caller
// because the top sections slice can't be reassigned through this
// helper.
func removeBlockByID(blocks []any, id string) bool {
	for _, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		children, hasChildren := b["children"].([]any)
		if hasChildren {
			for i, craw := range children {
				cb, ok := craw.(map[string]any)
				if !ok {
					continue
				}
				if cid, _ := cb["id"].(string); cid == id {
					b["children"] = append(children[:i], children[i+1:]...)
					return true
				}
			}
			if removeBlockByID(children, id) {
				return true
			}
		}
	}
	return false
}
