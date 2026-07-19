package store

import (
	"context"
	"sort"
	"time"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// rrfK is the reciprocal-rank-fusion constant. Larger flattens the
// contribution of rank position; 60 is the conventional default.
const rrfK = 60

// candidate is one ranked hit from a single retrieval channel (FTS or vector)
// before fusion. Backends return candidates already ordered best-first — FTS
// per type by descending lexical rank, vector by descending cosine similarity —
// and runHybridSearch turns each channel's ordering into reciprocal-rank
// scores. Only the ordering is consumed for fusion, never the raw scores, which
// is what lets one fusion serve every SQL dialect (approach Y).
type candidate struct {
	Type      string
	ID        string
	Title     string
	Excerpt   string
	Project   *string
	UpdatedAt time.Time
	// Similarity is the cosine similarity (0–1, = 1 - distance) of the best
	// matching chunk. Set on vector candidates, nil on FTS candidates. It
	// drives the relevance floor and surfaces on the resulting SearchHit.
	Similarity *float64
}

// searchBackend is the per-dialect half of hybrid search. Each backend produces
// ranked FTS and vector candidates (with its own excerpt highlighting); the
// dialect-free runHybridSearch fuses them identically across backends.
type searchBackend interface {
	// ftsCandidates returns full-text matches for the requested types, ordered
	// per type by descending lexical rank (best first within each type), each
	// with a highlighted excerpt. No limit — fusion caps the result.
	ftsCandidates(ctx context.Context, q string, types []string, f SearchFilter) ([]candidate, error)
	// vectorCandidates returns the best chunk per live source for the query
	// vector, ordered by descending cosine similarity, each carrying its
	// similarity. Orphaned vectors (source deleted) are excluded. No floor and
	// no limit — fusion applies both.
	vectorCandidates(ctx context.Context, vec []float32, types []string, f SearchFilter) ([]candidate, error)
	// fillPublicIDs populates each hit's PublicID from its source table, so a
	// non-document result can be deep-linked. Types without a public id (memory)
	// are left empty. Called once on the final page, so lookups stay bounded.
	fillPublicIDs(ctx context.Context, hits []*models.SearchHit) error
}

// publicIDTable maps a searchable type to its base table and whether that table
// has a public_id column (memory does not). Shared by both backends' fillPublicIDs.
func publicIDTable(typ string) (table string, hasPublicID bool) {
	switch typ {
	case "documents":
		return "documents", true
	case "decisions":
		return "decisions", true
	case "snippets":
		return "snippets", true
	case "solutions":
		return "solutions", true
	case "journal":
		return "journal_entries", true
	default: // memory, or unknown
		return "", false
	}
}

// runHybridSearch is the dialect-free core of unified search. It asks the
// backend for FTS candidates (always) and vector candidates (only when
// f.Vector is set), reciprocal-rank-fuses the two channels, applies the
// relevance floor to the vector side, and returns the top f.Limit hits.
//
// Fusion sums each side's reciprocal rank 1/(k+rank). Both channels rank
// globally over their backend-supplied ordering (FTS by lexical rank, vector
// by similarity), so two different types' top hits get distinct scores rather
// than both scoring 1/(k+1) (fusion normalization, road-p4-t4).
//
// maxDist is the cosine-distance floor (<= 0 disables it): a vector candidate
// survives only when its distance (1 - similarity) is strictly below maxDist,
// and dropped candidates never consume a rank position. The FTS side is never
// floored.
func runHybridSearch(ctx context.Context, b searchBackend, q string, types []string, f SearchFilter, maxDist float64) ([]*models.SearchHit, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	fts, err := b.ftsCandidates(ctx, q, types, f)
	if err != nil {
		return nil, err
	}

	// key is (type, id): a source is one fused hit regardless of channel.
	type key struct{ typ, id string }
	acc := map[key]*models.SearchHit{}
	var order []*models.SearchHit

	upsert := func(c candidate) *models.SearchHit {
		k := key{c.Type, c.ID}
		h, ok := acc[k]
		if !ok {
			// First channel to reach a source projects its display fields;
			// the FTS side runs first, so it wins for a hit in both channels.
			h = &models.SearchHit{
				Type: c.Type, ID: c.ID, Title: c.Title, Excerpt: c.Excerpt,
				Project: c.Project, UpdatedAt: c.UpdatedAt,
			}
			acc[k] = h
			order = append(order, h)
		}
		return h
	}

	// FTS side: global reciprocal rank over the backend's lexical-rank order,
	// so cross-type top hits get distinct scores (road-p4-t4).
	ftsRank := 0
	for _, c := range fts {
		ftsRank++
		upsert(c).Score += 1.0 / float64(rrfK+ftsRank)
	}

	if f.Vector != nil {
		vec, err := b.vectorCandidates(ctx, f.Vector, types, f)
		if err != nil {
			return nil, err
		}
		rank := 0
		for _, c := range vec {
			// Relevance floor: drop candidates at/beyond the distance
			// threshold before they take a rank slot.
			if maxDist > 0 && c.Similarity != nil && (1-*c.Similarity) >= maxDist {
				continue
			}
			rank++
			h := upsert(c)
			h.Score += 1.0 / float64(rrfK+rank)
			h.Similarity = c.Similarity // surface the semantic relevance
		}
	}

	// Highest fused score first; recency then a stable id/type tiebreak keep
	// the ordering deterministic (the original SQL left score+recency ties
	// unordered).
	sort.Slice(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	})
	if len(order) > limit {
		order = order[:limit]
	}
	// Attach each hit's public id (for deep-linking) on the final page only.
	if err := b.fillPublicIDs(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}
