package store

import (
	"context"
	"testing"
	"time"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// fakeBackend feeds runHybridSearch canned candidate lists so the fusion core
// can be exercised with no database.
type fakeBackend struct {
	fts, vec []candidate
	ftsErr   error
	vecErr   error
}

func (b fakeBackend) ftsCandidates(_ context.Context, _ string, _ []string, _ SearchFilter) ([]candidate, error) {
	return b.fts, b.ftsErr
}

func (b fakeBackend) vectorCandidates(_ context.Context, _ []float32, _ []string, _ SearchFilter) ([]candidate, error) {
	return b.vec, b.vecErr
}

// fillPublicIDs is a no-op for the fusion unit tests (no backing tables).
func (b fakeBackend) fillPublicIDs(_ context.Context, _ []*models.SearchHit) error { return nil }

func simPtr(v float64) *float64 { return &v }

// fc builds an FTS candidate (no similarity). ts orders the deterministic
// tiebreak on equal scores.
func fc(typ, id string, ts int) candidate {
	return candidate{Type: typ, ID: id, Title: id, Excerpt: id + " body",
		UpdatedAt: time.Unix(int64(ts), 0)}
}

// vc builds a vector candidate carrying a cosine similarity.
func vc(typ, id string, sim float64) candidate {
	return candidate{Type: typ, ID: id, Title: id, Excerpt: id + " chunk",
		UpdatedAt: time.Unix(0, 0), Similarity: simPtr(sim)}
}

func ids(hits []*models.SearchHit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.ID
	}
	return out
}

func rr(rank int) float64 { return 1.0 / float64(rrfK+rank) }

// A doc reachable through both channels outscores docs reached through only
// one, and the fused order reflects the summed reciprocal ranks.
func TestRunHybridSearchFusesChannelsByReciprocalRank(t *testing.T) {
	b := fakeBackend{
		fts: []candidate{fc("documents", "A", 2), fc("documents", "B", 1)},
		vec: []candidate{vc("documents", "B", 0.9), vc("documents", "C", 0.8)},
	}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Vector: []float32{1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A: fts rank1 = rr(1). B: fts rank2 + vec rank1 = rr(2)+rr(1). C: vec rank2 = rr(2).
	// B (0.0325) > A (0.0164) > C (0.0161).
	if got := ids(hits); len(got) != 3 || got[0] != "B" || got[1] != "A" || got[2] != "C" {
		t.Fatalf("fused order = %v, want [B A C]", got)
	}
	wantB := rr(2) + rr(1)
	if hits[0].Score != wantB {
		t.Errorf("B score = %v, want %v", hits[0].Score, wantB)
	}
	// B carries the vector-side similarity; A (FTS-only) carries none.
	if hits[0].Similarity == nil || *hits[0].Similarity != 0.9 {
		t.Errorf("B similarity = %v, want 0.9", hits[0].Similarity)
	}
	if hits[1].Similarity != nil {
		t.Errorf("A (FTS-only) similarity = %v, want nil", hits[1].Similarity)
	}
}

// FTS-side projection (title/excerpt) wins over the vector side for a hit
// present in both channels.
func TestRunHybridSearchFTSProjectionWins(t *testing.T) {
	ftsCand := candidate{Type: "documents", ID: "A", Title: "FTS title", Excerpt: "fts excerpt", UpdatedAt: time.Unix(5, 0)}
	vecCand := candidate{Type: "documents", ID: "A", Title: "VEC title", Excerpt: "vec excerpt", UpdatedAt: time.Unix(9, 0), Similarity: simPtr(0.7)}
	b := fakeBackend{fts: []candidate{ftsCand}, vec: []candidate{vecCand}}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Vector: []float32{1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1", len(hits))
	}
	if hits[0].Title != "FTS title" || hits[0].Excerpt != "fts excerpt" {
		t.Errorf("projection = %q/%q, want FTS side to win", hits[0].Title, hits[0].Excerpt)
	}
	if hits[0].Similarity == nil || *hits[0].Similarity != 0.7 {
		t.Errorf("similarity = %v, want 0.7 from vector side", hits[0].Similarity)
	}
}

// With no query vector, only FTS candidates surface and no similarity is set —
// the vector producer is never consulted.
func TestRunHybridSearchFTSOnlyWhenNoVector(t *testing.T) {
	b := fakeBackend{
		fts: []candidate{fc("documents", "A", 2), fc("documents", "B", 1)},
		vec: []candidate{vc("documents", "Z", 1.0)}, // must be ignored
	}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(hits); len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("FTS-only order = %v, want [A B]", got)
	}
	for _, h := range hits {
		if h.Similarity != nil {
			t.Errorf("FTS-only hit %s carries similarity %v, want nil", h.ID, *h.Similarity)
		}
	}
}

// The relevance floor drops vector candidates beyond the cosine-distance
// threshold before they consume a rank position; survivors keep contiguous
// reciprocal ranks.
func TestRunHybridSearchAppliesVectorFloor(t *testing.T) {
	b := fakeBackend{
		vec: []candidate{vc("documents", "near", 1.0), vc("documents", "far", 0.0)},
	}
	// maxDist 0.45: near (distance 0) stays, far (distance 1.0) is dropped.
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Vector: []float32{1}}, 0.45)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(hits); len(got) != 1 || got[0] != "near" {
		t.Fatalf("floored result = %v, want [near]", got)
	}
	// near survived as the first vector candidate → reciprocal rank 1.
	if hits[0].Score != rr(1) {
		t.Errorf("near score = %v, want rr(1)=%v", hits[0].Score, rr(1))
	}
}

// A dropped floor candidate does not consume a rank slot: a following survivor
// still scores as rank 1, not rank 2.
func TestRunHybridSearchFloorDoesNotConsumeRank(t *testing.T) {
	b := fakeBackend{
		vec: []candidate{vc("documents", "far", 0.0), vc("documents", "near", 1.0)},
	}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Vector: []float32{1}}, 0.45)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(hits); len(got) != 1 || got[0] != "near" {
		t.Fatalf("result = %v, want [near]", got)
	}
	if hits[0].Score != rr(1) {
		t.Errorf("near score = %v, want rr(1)=%v (far must not consume rank 1)", hits[0].Score, rr(1))
	}
}

// The final limit caps the fused result.
func TestRunHybridSearchAppliesLimit(t *testing.T) {
	b := fakeBackend{fts: []candidate{
		fc("documents", "A", 3), fc("documents", "B", 2), fc("documents", "C", 1)}}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Limit: 2}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(hits); len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("limited result = %v, want [A B]", got)
	}
}

// Fusion is normalized across content types (road-p4-t4): FTS reciprocal rank
// is global, so two different types' top hits get distinct scores instead of
// both scoring rr(1). The backend returns FTS candidates in global lexical-rank
// order; runHybridSearch scores them by global position.
func TestRunHybridSearchRanksFTSGloballyAcrossTypes(t *testing.T) {
	b := fakeBackend{fts: []candidate{
		fc("documents", "A", 9), fc("decisions", "X", 8), fc("documents", "B", 7)}}
	hits, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents", "decisions"}, SearchFilter{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"A": rr(1), "X": rr(2), "B": rr(3)}
	for _, h := range hits {
		if h.Score != want[h.ID] {
			t.Errorf("%s score = %v, want %v (global RRF)", h.ID, h.Score, want[h.ID])
		}
	}
	// The two types' top hits (A: documents, X: decisions) must not tie.
	if got := ids(hits); got[0] != "A" || got[1] != "X" {
		t.Errorf("global order = %v, want [A X B]", got)
	}
	if hits[0].Score == hits[1].Score {
		t.Errorf("cross-type top hits tied at %v; fusion not normalized", hits[0].Score)
	}
}

// An error from either candidate producer propagates.
func TestRunHybridSearchPropagatesProducerErrors(t *testing.T) {
	sentinel := context.DeadlineExceeded
	b := fakeBackend{ftsErr: sentinel}
	if _, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{}, 0); err != sentinel {
		t.Fatalf("fts error = %v, want propagation", err)
	}
	b = fakeBackend{vecErr: sentinel}
	if _, err := runHybridSearch(context.Background(), b, "q",
		[]string{"documents"}, SearchFilter{Vector: []float32{1}}, 0); err != sentinel {
		t.Fatalf("vec error = %v, want propagation", err)
	}
}
