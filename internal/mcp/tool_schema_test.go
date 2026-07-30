package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestToolsAdvertiseMinimalOutputSchemas(t *testing.T) {
	cs := newClient(t)
	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) == 0 {
		t.Fatal("server advertised no tools")
	}

	const want = `{"type":"object"}`
	for _, tool := range result.Tools {
		got, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatalf("marshal %s output schema: %v", tool.Name, err)
		}
		if string(got) != want {
			t.Errorf("%s output schema = %s, want %s", tool.Name, got, want)
		}
	}
}

func TestInstructionsUseContextBundleAsTheSingleStartupRead(t *testing.T) {
	cs := newClient(t)
	instructions := cs.InitializeResult().Instructions
	if !strings.Contains(instructions, "Session start: call get_context_bundle") {
		t.Fatal("instructions do not direct startup through get_context_bundle")
	}

	// No per-type startup reads: the deleted readers must not resurface.
	for _, gone := range []string{
		"get_memory",
		"get_journal",
		"get_decisions",
		"query_decisions",
		"search_documents",
		"get_snippets",
		"find_solution",
		"create_project",
	} {
		if strings.Contains(instructions, gone) {
			t.Errorf("instructions reference deleted tool %q", gone)
		}
	}
}
