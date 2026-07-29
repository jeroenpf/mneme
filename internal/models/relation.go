package models

import "time"

// Relation is one directed edge in the polymorphic relations graph
// (spec-relations). Endpoints are public ids — the prefix encodes the kind.
// ToRef keeps the reference exactly as written (a public id or a wikilink
// slug) so a dangling reference can resolve at query time once its target
// exists; ToID is the resolved public id, nil while dangling.
type Relation struct {
	ID        int64     `json:"id"`
	FromID    string    `json:"from_id"`
	ToRef     string    `json:"to_ref"`
	ToID      *string   `json:"to_id,omitempty"`
	RelType   string    `json:"rel_type"` // mentions | relates-to | implements | supersedes | depends-on | blocks
	Origin    string    `json:"origin"`   // auto | explicit
	CreatedAt time.Time `json:"created_at"`
}
