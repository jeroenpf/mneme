package ids

import "fmt"

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

// nestedRef formats mneme://document/<docID>/<kind>/<childID>, validating
// that the owner is a document id and the child matches the nested kind.
func nestedRef(docID string, kind Kind, childID string) (string, error) {
	if !ValidFor(KindDocument, docID) {
		return "", fmt.Errorf("ids: %q is not a valid document id", docID)
	}
	if !ValidFor(kind, childID) {
		return "", fmt.Errorf("ids: %q is not a valid %s id", childID, kind)
	}
	return scheme + string(KindDocument) + "/" + docID + "/" + string(kind) + "/" + childID, nil
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
