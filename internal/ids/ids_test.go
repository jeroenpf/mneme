package ids_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/ids"
)

// allKinds is every addressable entity kind, used to assert the prefix
// registry and generator cover the whole vocabulary.
var allKinds = []struct {
	kind   ids.Kind
	prefix string
}{
	{ids.KindProject, "prj_"},
	{ids.KindDocument, "doc_"},
	{ids.KindBlock, "blk_"},
	{ids.KindTask, "task_"},
	{ids.KindDecision, "dec_"},
	{ids.KindJournal, "jrnl_"},
	{ids.KindSnippet, "snip_"},
	{ids.KindSolution, "sol_"},
}

func TestNewCarriesTheKindsPrefixAndAValidBody(t *testing.T) {
	for _, tc := range allKinds {
		id, err := ids.New(tc.kind)
		if err != nil {
			t.Fatalf("New(%s): %v", tc.kind, err)
		}
		if !strings.HasPrefix(id, tc.prefix) {
			t.Errorf("New(%s) = %q, want prefix %q", tc.kind, id, tc.prefix)
		}
		if !ids.Valid(id) {
			t.Errorf("New(%s) = %q is not Valid", tc.kind, id)
		}
		gotKind, _, err := ids.Parse(id)
		if err != nil {
			t.Errorf("Parse(%q): %v", id, err)
		}
		if gotKind != tc.kind {
			t.Errorf("Parse(%q) kind = %s, want %s", id, gotKind, tc.kind)
		}
	}
}

func TestNewRejectsAnUnknownKind(t *testing.T) {
	if _, err := ids.New(ids.Kind("banana")); err == nil {
		t.Fatal("New(banana) should error on an unregistered kind")
	}
}

func TestNewIsDeterministicForAFixedReader(t *testing.T) {
	// 12 zero bytes map to the first alphabet symbol repeated.
	g := ids.NewGen(bytes.NewReader(make([]byte, 12)))
	id, err := g.New(ids.KindBlock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if want := "blk_000000000000"; id != want {
		t.Fatalf("deterministic New = %q, want %q", id, want)
	}
}

func TestNewSurfacesAReaderError(t *testing.T) {
	g := ids.NewGen(bytes.NewReader(nil)) // empty: ReadFull hits EOF
	if _, err := g.New(ids.KindBlock); err == nil {
		t.Fatal("New should surface a random-source read error")
	}
}

func TestGeneratedIDsAreDistinct(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 2000; i++ {
		id, err := ids.New(ids.KindTask)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if seen[id] {
			t.Fatalf("collision after %d ids: %q", i, id)
		}
		seen[id] = true
	}
}

func TestParseRejectsMalformedIDs(t *testing.T) {
	cases := map[string]string{
		"no separator":       "blk000000000000",
		"unknown prefix":     "xyz_000000000000",
		"empty body":         "blk_",
		"short body":         "blk_0000",
		"long body":          "blk_0000000000000",
		"crockford-excluded": "blk_00000000000i", // i is not in the alphabet
		"uppercase":          "blk_00000000000A",
		"empty":              "",
	}
	for name, id := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ids.Parse(id); err == nil {
				t.Errorf("Parse(%q) should error", id)
			}
			if ids.Valid(id) {
				t.Errorf("Valid(%q) should be false", id)
			}
		})
	}
}

func TestValidForChecksPrefixMatchesKind(t *testing.T) {
	block, _ := ids.New(ids.KindBlock)
	if !ids.ValidFor(ids.KindBlock, block) {
		t.Errorf("ValidFor(block, %q) = false, want true", block)
	}
	if ids.ValidFor(ids.KindDocument, block) {
		t.Errorf("ValidFor(document, %q) = true, want false (prefix mismatch)", block)
	}
	if ids.ValidFor(ids.KindBlock, "not-an-id") {
		t.Errorf("ValidFor(block, malformed) = true, want false")
	}
}

func TestNewUniqueRetriesPastCollisions(t *testing.T) {
	// First draw (12 zero bytes) collides; second draw (12 one bytes) is fresh.
	buf := append(make([]byte, 12), bytes.Repeat([]byte{1}, 12)...)
	g := ids.NewGen(bytes.NewReader(buf))
	taken := map[string]bool{"blk_000000000000": true}
	id, err := g.NewUnique(ids.KindBlock, func(candidate string) bool { return taken[candidate] })
	if err != nil {
		t.Fatalf("NewUnique: %v", err)
	}
	if id == "blk_000000000000" {
		t.Fatal("NewUnique returned the colliding id instead of retrying")
	}
	if id != "blk_111111111111" {
		t.Fatalf("NewUnique = %q, want the second draw blk_111111111111", id)
	}
}

func TestNewUniqueGivesUpWhenEverythingCollides(t *testing.T) {
	g := ids.NewGen(bytes.NewReader(bytes.Repeat([]byte{0}, 512)))
	_, err := g.NewUnique(ids.KindBlock, func(string) bool { return true })
	if err == nil {
		t.Fatal("NewUnique should error when no candidate is ever free")
	}
}

func TestNewUniqueToleratesANilPredicate(t *testing.T) {
	id, err := ids.NewUnique(ids.KindSnippet, nil)
	if err != nil {
		t.Fatalf("NewUnique(nil): %v", err)
	}
	if !ids.ValidFor(ids.KindSnippet, id) {
		t.Fatalf("NewUnique(nil) = %q, not a valid snippet id", id)
	}
}
