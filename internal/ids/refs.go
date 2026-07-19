package ids

import (
	"fmt"
	"strings"
)

// scheme is the canonical reference prefix. A reference is a self-describing
// URI so an agent can recognise and resolve a pasted id without inferring the
// grammar: mneme://<kind>/<id>, with blocks and tasks nested under their
// owning document.
const scheme = "mneme://"

// Ref formats the canonical reference for a top-level entity, rejecting an
// id whose prefix does not match kind. Block and task references are nested
// under a document, so Ref rejects those kinds — use RefBlock / RefTask.
func Ref(kind Kind, id string) (string, error) {
	if kind == KindBlock || kind == KindTask {
		return "", fmt.Errorf("ids: %s references nest under a document; use Ref%s", kind, titleKind(kind))
	}
	if !ValidFor(kind, id) {
		return "", fmt.Errorf("ids: %q is not a valid %s id", id, kind)
	}
	return scheme + string(kind) + "/" + id, nil
}

// RefBlock formats the canonical reference for a block owned by a document.
func RefBlock(docID, blockID string) (string, error) {
	return nestedRef(docID, KindBlock, blockID)
}

// RefTask formats the canonical reference for a task owned by a document.
func RefTask(docID, taskID string) (string, error) {
	return nestedRef(docID, KindTask, taskID)
}

// nestedRef formats mneme://document/<docID>/<kind>/<childID>. The owner must
// be a document public id; the child id is document-local, so it may be a
// generated blk_/task_ id or a legacy semantic id — any non-empty, slash-free
// token. The relation (kind), not the child's prefix, marks it a block vs task.
func nestedRef(docID string, kind Kind, childID string) (string, error) {
	if !ValidFor(KindDocument, docID) {
		return "", fmt.Errorf("ids: %q is not a valid document id", docID)
	}
	if childID == "" || strings.ContainsRune(childID, '/') {
		return "", fmt.Errorf("ids: %q is not a valid %s id", childID, kind)
	}
	return scheme + string(KindDocument) + "/" + docID + "/" + string(kind) + "/" + childID, nil
}

// Reference is a parsed mneme:// reference. For a top-level entity, Kind and ID
// identify it and DocID is empty. For a nested block or task reference, Kind is
// KindBlock or KindTask, ID is the child id, and DocID is the owning document.
type Reference struct {
	Kind  Kind
	ID    string
	DocID string
}

// ParseRef parses a canonical mneme:// reference into its components — the exact
// inverse of Ref, RefBlock, and RefTask. It rejects a wrong scheme, an unknown
// or nested-only kind segment, a prefix that does not match its kind, a
// malformed id, a nested reference not owned by a document, an unnestable
// relation, and any missing, trailing, or extra segments.
func ParseRef(ref string) (Reference, error) {
	if !strings.HasPrefix(ref, scheme) {
		return Reference{}, fmt.Errorf("ids: %q is not a mneme:// reference", ref)
	}
	segs := strings.Split(strings.TrimPrefix(ref, scheme), "/")
	switch len(segs) {
	case 2: // mneme://<kind>/<id>
		kind, id := Kind(segs[0]), segs[1]
		if kind == KindBlock || kind == KindTask {
			return Reference{}, fmt.Errorf("ids: %s references nest under a document; expected mneme://document/…/%s/…", kind, kind)
		}
		if _, ok := kindPrefix[kind]; !ok {
			return Reference{}, fmt.Errorf("ids: %q has an unknown entity kind %q", ref, segs[0])
		}
		if !ValidFor(kind, id) {
			return Reference{}, fmt.Errorf("ids: %q is not a valid %s id", id, kind)
		}
		return Reference{Kind: kind, ID: id}, nil
	case 4: // mneme://document/<docID>/<relation>/<childID>
		if Kind(segs[0]) != KindDocument {
			return Reference{}, fmt.Errorf("ids: nested references must be owned by a document, got %q", segs[0])
		}
		docID, child, childID := segs[1], Kind(segs[2]), segs[3]
		if child != KindBlock && child != KindTask {
			return Reference{}, fmt.Errorf("ids: %q is not a nestable relation; expected block or task", segs[2])
		}
		if !ValidFor(KindDocument, docID) {
			return Reference{}, fmt.Errorf("ids: %q is not a valid document id", docID)
		}
		// The child id is document-local: a generated blk_/task_ id or a legacy
		// semantic id. Accept any non-empty token (a trailing-slash ref lands
		// here with an empty childID and is rejected).
		if childID == "" {
			return Reference{}, fmt.Errorf("ids: %q is missing its %s id", ref, child)
		}
		return Reference{Kind: child, ID: childID, DocID: docID}, nil
	default:
		return Reference{}, fmt.Errorf("ids: %q is not a well-formed mneme reference", ref)
	}
}

// titleKind returns the exported helper name a caller should use for a
// nested kind, for the redirect message: "Block" or "Task".
func titleKind(kind Kind) string {
	switch kind {
	case KindBlock:
		return "Block"
	case KindTask:
		return "Task"
	default:
		return string(kind)
	}
}
