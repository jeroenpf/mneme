package store

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestToFTS5QueryShape checks the websearch→FTS5 translator maps the google-style
// operators (implicit AND, quoted phrase, `or`, `-exclusion`) and reports ok=false
// for queries with no positive term.
func TestToFTS5QueryShape(t *testing.T) {
	cases := []struct {
		in     string
		wantOK bool
	}{
		{"", false},
		{"   ", false},
		{"hello", true},
		{"hello world", true},
		{`"hello world"`, true},
		{"go or rust", true},
		{"web -test", true},
		{"-test", false},           // only an exclusion → no positive term
		{"((()))", false},          // no searchable tokens
		{"a b or c", true},         // (a AND b) OR c
		{`"unclosed`, true},        // tolerate a dangling quote
		{"a AND b", true},          // AND is a literal word here, not an operator
		{"café münchen", true},     // unicode letters count as searchable
	}
	for _, c := range cases {
		got, ok := toFTS5Query(c.in)
		if ok != c.wantOK {
			t.Errorf("toFTS5Query(%q): ok=%v want %v (out=%q)", c.in, ok, c.wantOK, got)
		}
		if ok && got == "" {
			t.Errorf("toFTS5Query(%q): ok but empty output", c.in)
		}
	}
}

// TestToFTS5QueryNeverErrors is the critical safety guarantee: no input — however
// malformed — may translate to a string FTS5 rejects as a syntax error. Every
// translated query is run through a real fts5 MATCH.
func TestToFTS5QueryNeverErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE VIRTUAL TABLE ft USING fts5(body)`); err != nil {
		t.Fatalf("create fts5: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO ft(body) VALUES ('the quick brown fox jumps')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	adversarial := []string{
		"", " ", "\t\n", "hello", "hello world", `"phrase here"`,
		"a or b", "-only", "a -b -c", `un"closed`, `"`, `""`, `"" ""`,
		"* ( ) : ^ NEAR", "AND OR NOT", "foo* bar", "a:b c:d",
		"-\"neg phrase\"", "多字节 test", "  or  or  ", "( AND )",
		`quote " inside`, "trailing-", "-", "verylongtoken" + string(make([]byte, 0)),
		"'apostrophe's'", "semi;colon", "back\\slash", "new\nline term",
	}
	for _, in := range adversarial {
		q, ok := toFTS5Query(in)
		if !ok {
			continue // skipped queries never touch FTS5
		}
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ft WHERE ft MATCH ?`, q).Scan(&n); err != nil {
			t.Errorf("input %q → MATCH %q raised FTS5 error: %v", in, q, err)
		}
	}
}

// TestCosineSim spot-checks the brute-force cosine used by vectorCandidates.
func TestCosineSim(t *testing.T) {
	if got := cosineSim([]float32{1, 0}, []float32{1, 0}); got < 0.999 || got > 1.001 {
		t.Errorf("identical vectors: got %v want ~1", got)
	}
	if got := cosineSim([]float32{1, 0}, []float32{0, 1}); got < -0.001 || got > 0.001 {
		t.Errorf("orthogonal vectors: got %v want ~0", got)
	}
	if got := cosineSim([]float32{1, 0}, []float32{-1, 0}); got > -0.999 {
		t.Errorf("opposite vectors: got %v want ~-1", got)
	}
	if got := cosineSim(nil, nil); got != 0 {
		t.Errorf("empty vectors: got %v want 0", got)
	}
}
