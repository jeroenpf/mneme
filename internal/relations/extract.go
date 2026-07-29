// Package relations implements the polymorphic relations graph
// (spec-relations): reference extraction from document bodies, the typed
// link service, and the startup backfill.
package relations

import (
	"regexp"
	"sort"

	"github.com/jeroenpf/mneme/internal/models"
)

var (
	// rePublicID matches the top-level entity kinds that can be relation
	// targets. blk_/task_/proj_ are deliberately absent: a mention of a
	// block or task is a mention of its document, which is nearly always
	// referenced alongside it.
	rePublicID = regexp.MustCompile(`\b(?:doc|dec|snip|sol|jrnl)_[a-z0-9]{6,}\b`)
	reWikilink = regexp.MustCompile(`\[\[([A-Za-z0-9][A-Za-z0-9_-]*)\]\]`)
)

// ExtractRefs returns every foreign reference in the document's body —
// entity public ids and [[slug]] wikilink targets — deduped, sorted, with
// self-references (the doc's own public id and slug) removed. All body
// strings are scanned, code and diagram content included: a reference in a
// code comment is still a reference.
func ExtractRefs(doc *models.Document) []string {
	seen := map[string]bool{}
	walkStrings(doc.Body, func(s string) {
		for _, m := range rePublicID.FindAllString(s, -1) {
			seen[m] = true
		}
		for _, m := range reWikilink.FindAllStringSubmatch(s, -1) {
			seen[m[1]] = true
		}
	})
	delete(seen, doc.PublicID)
	delete(seen, doc.ID)
	out := make([]string, 0, len(seen))
	for r := range seen {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// MentionRows converts the document's extracted refs into origin='auto'
// mention rows ready for Store.ReplaceAutoMentions. ToID is set when the
// ref is already a public id; wikilink slugs stay dangling and resolve at
// query time.
func MentionRows(doc *models.Document) []*models.Relation {
	refs := ExtractRefs(doc)
	rows := make([]*models.Relation, 0, len(refs))
	for _, ref := range refs {
		r := &models.Relation{FromID: doc.PublicID, ToRef: ref, RelType: "mentions", Origin: "auto"}
		if rePublicID.MatchString(ref) {
			id := ref
			r.ToID = &id
		}
		rows = append(rows, r)
	}
	return rows
}

func walkStrings(v any, fn func(string)) {
	switch x := v.(type) {
	case string:
		fn(x)
	case map[string]any:
		for _, val := range x {
			walkStrings(val, fn)
		}
	case []any:
		for _, val := range x {
			walkStrings(val, fn)
		}
	}
}
