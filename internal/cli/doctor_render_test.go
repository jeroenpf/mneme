package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCountFailed(t *testing.T) {
	results := []check{
		ok("a", ""), warn("b", ""), fail("c", ""), fail("d", ""),
	}
	if got := countFailed(results); got != 2 {
		t.Errorf("countFailed: got %d, want 2", got)
	}
	if got := countFailed([]check{ok("a", ""), warn("b", "")}); got != 0 {
		t.Errorf("no failures: got %d, want 0", got)
	}
}

// The scorecard names every check and summarizes the outcome.
func TestRenderScorecard(t *testing.T) {
	var buf bytes.Buffer
	renderScorecard(&buf, []check{
		ok("config", "SQLite backend"),
		fail("database", "not reachable"),
	})
	out := buf.String()
	for _, want := range []string{"mneme doctor", "config", "SQLite backend", "database", "not reachable", "1 check(s) failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("scorecard missing %q; got:\n%s", want, out)
		}
	}

	var okBuf bytes.Buffer
	renderScorecard(&okBuf, []check{ok("config", "fine")})
	if !strings.Contains(okBuf.String(), "All checks passed") {
		t.Errorf("all-OK scorecard should say so; got:\n%s", okBuf.String())
	}
}
