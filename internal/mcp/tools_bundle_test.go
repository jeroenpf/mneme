package mcp_test

import (
	"strings"
	"testing"
)

func TestGetContextBundle(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	// seed content through the existing tools so the bundle has data
	call(t, cs, "set_memory", map[string]any{
		"scope": "project", "project": "apollo", "key": "db", "value": "postgres",
	}, nil)
	call(t, cs, "log_decision", map[string]any{
		"project": "apollo", "title": "use pgx", "decision": "pgx/v5",
	}, nil)
	call(t, cs, "set_env", map[string]any{
		"project": "apollo", "key": "API_PORT", "value": "8443",
	}, nil)

	// The MCP handler returns only the pre-rendered markdown digest — no
	// structured project/memory/decisions/env fields.
	var m map[string]any
	call(t, cs, "get_context_bundle", map[string]any{"project": "apollo"}, &m)
	md, _ := m["markdown"].(string)
	if md == "" {
		t.Errorf("bundle markdown empty")
	}
	if len(m) != 1 {
		t.Errorf("bundle should return only markdown, got keys %v", m)
	}
	// The digest still carries everything that was assembled.
	if !strings.Contains(md, "use pgx") {
		t.Errorf("digest lost the assembled decision: %q", md)
	}
}

func TestGetContextBundleBudget(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	for _, title := range []string{"alpha choice", "beta choice", "gamma choice", "delta choice"} {
		call(t, cs, "log_decision", map[string]any{
			"project": "apollo", "title": title, "decision": "x",
			"rationale": "a reasonably long rationale that consumes budget so trimming becomes observable once the cap is tight",
		}, nil)
	}

	var big, small map[string]any
	call(t, cs, "get_context_bundle", map[string]any{"project": "apollo"}, &big)
	call(t, cs, "get_context_bundle", map[string]any{"project": "apollo", "budget": 60}, &small)
	bigMD, _ := big["markdown"].(string)
	smallMD, _ := small["markdown"].(string)

	if len(smallMD) >= len(bigMD) {
		t.Errorf("tight budget should shrink the digest: small=%d big=%d", len(smallMD), len(bigMD))
	}
	if strings.Count(smallMD, "choice") >= strings.Count(bigMD, "choice") {
		t.Errorf("tight budget should drop decisions:\nsmall:\n%s\nbig:\n%s", smallMD, bigMD)
	}
}

func TestGetContextBundleUnknownProject(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "get_context_bundle", map[string]any{"project": "ghost"}); msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestGetContextBundleMissingProject(t *testing.T) {
	cs := newClient(t)
	if msg := callExpectError(t, cs, "get_context_bundle", map[string]any{}); msg == "" {
		t.Error("expected project-required error")
	}
}
