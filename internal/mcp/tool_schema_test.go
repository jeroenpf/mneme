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
	if !strings.Contains(instructions, "At the start of every session, call get_context_bundle") {
		t.Fatal("instructions do not direct startup through get_context_bundle")
	}

	for _, duplicate := range []string{
		"At session start, call get_memory",
		"call get_journal at the start",
	} {
		if strings.Contains(instructions, duplicate) {
			t.Errorf("instructions retain duplicate startup read %q", duplicate)
		}
	}
}
