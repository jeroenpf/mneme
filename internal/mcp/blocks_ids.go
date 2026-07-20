package mcp

import (
	"fmt"

	"github.com/jeroenpf/mneme/internal/ids"
)

// The architecture contract is "all id fields must be unique within a
// document", but validateBody only checks block types and field names — it
// never enforced that blocks and tasks actually carry an id, let alone a
// unique one. Duplicate or missing ids silently break the id-addressed edit
// tools (tick_task, update_section, …) and the block-level live flash, which
// all resolve the FIRST match. This file closes that gap: Mneme mints ids for
// blocks and tasks that omit them and rejects any that are non-string or
// duplicated, so the store is authoritative for nested identity.

// idNode is a block or task reached while walking a document body, tagged
// with the kind of public id it should carry. node is the live map, so
// assigning node["id"] mutates the document in place.
type idNode struct {
	node map[string]any
	kind ids.Kind
	path string
}

// createdID is one server-minted id, reported back so a caller learns the
// stable handle for a block or task it did not name.
type createdID struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`            // "block" | "task"
	Label string `json:"label,omitempty"` // block title/type or task title, for orientation
}

// collectIDNodes appends every block and task under blocks — recursing
// through children and task arrays — to *out, in document order.
func collectIDNodes(blocks []any, path string, out *[]idNode) error {
	for i, raw := range blocks {
		b, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", path, i)
		}
		p := fmt.Sprintf("%s[%d]", path, i)
		*out = append(*out, idNode{node: b, kind: ids.KindBlock, path: p})

		if tasks, ok := b["tasks"].([]any); ok {
			for j, traw := range tasks {
				tm, ok := traw.(map[string]any)
				if !ok {
					return fmt.Errorf("%s.tasks[%d] must be an object", p, j)
				}
				*out = append(*out, idNode{node: tm, kind: ids.KindTask, path: fmt.Sprintf("%s.tasks[%d]", p, j)})
			}
		}
		if children, ok := b["children"].([]any); ok {
			if err := collectIDNodes(children, p+".children", out); err != nil {
				return err
			}
		}
	}
	return nil
}

// bodyIDNodes returns every block/task node in body.sections.
func bodyIDNodes(body map[string]any) ([]idNode, error) {
	sections, err := sectionsArray(body)
	if err != nil {
		return nil, err
	}
	var nodes []idNode
	if err := collectIDNodes(sections, "body.sections", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// resolveIDs registers each supplied id into taken and mints a fresh unique
// id for every node that omits one. A supplied id is rejected when it is
// non-string, or already present in taken — whether that is a duplicate
// within nodes or a collision with an id seeded from the rest of the
// document. taken is mutated with every id seen or minted. Returns the ids
// it minted, in document order.
func resolveIDs(nodes []idNode, taken map[string]bool) ([]createdID, error) {
	for _, n := range nodes {
		raw, present := n.node["id"]
		if !present {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("%s: id must be a string", n.path)
		}
		if s == "" {
			continue // treat an explicit empty id as omitted
		}
		if taken[s] {
			return nil, fmt.Errorf("%s: id %q is already used in this document — every block and task id must be unique", n.path, s)
		}
		taken[s] = true
	}

	var created []createdID
	for _, n := range nodes {
		if s, _ := n.node["id"].(string); s != "" {
			continue
		}
		newID, err := ids.NewUnique(n.kind, func(c string) bool { return taken[c] })
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n.path, err)
		}
		n.node["id"] = newID
		taken[newID] = true
		created = append(created, createdID{ID: newID, Kind: string(n.kind), Label: nodeLabel(n)})
	}
	return created, nil
}

// normalizeBodyIDs assigns a generated public id to every block and task in
// body that lacks one, preserves supplied ids, and rejects a non-string or
// duplicated id anywhere in the document. Returns the ids it minted.
func normalizeBodyIDs(body map[string]any) ([]createdID, error) {
	nodes, err := bodyIDNodes(body)
	if err != nil {
		return nil, err
	}
	return resolveIDs(nodes, map[string]bool{})
}

// existingIDSet collects every block/task id already present in body into a
// set. Duplicates that predate this rule are collapsed rather than rejected —
// the add_* tools only need to know which ids are taken, not to re-validate
// history.
func existingIDSet(body map[string]any) (map[string]bool, error) {
	nodes, err := bodyIDNodes(body)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if s, _ := n.node["id"].(string); s != "" {
			set[s] = true
		}
	}
	return set, nil
}

// nodeLabel returns a short human label for a created-id outline: a title if
// the node has one, else its block type (tasks have no type, so they fall
// back to the title).
func nodeLabel(n idNode) string {
	if t, _ := n.node["title"].(string); t != "" {
		return t
	}
	if t, _ := n.node["type"].(string); t != "" {
		return t
	}
	return ""
}
