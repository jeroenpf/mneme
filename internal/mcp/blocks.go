package mcp

import "fmt"

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

// validateBody walks body.sections recursively and returns an error on
// the first block with an unknown type. A section's children and a
// subphase's tasks are walked too — task entries are not blocks (they
// carry no type field), so they are not validated for type, only for
// shape.
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
		if children, ok := b["children"].([]any); ok {
			if err := walkBlocks(children, fmt.Sprintf("%s[%d].children", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
