package mcp_test

import (
	"context"
	"testing"
)

func TestSetMemoryEmptyValueDeletes(t *testing.T) {
	cs := newClient(t)

	// A normal set acks with just the key — no echo of the value.
	var setOut map[string]any
	call(t, cs, "set_memory", map[string]any{"scope": "global", "key": "diet-k", "value": "v"}, &setOut)
	if setOut["key"] != "diet-k" {
		t.Errorf("set ack key: got %v, want diet-k", setOut["key"])
	}
	if _, ok := setOut["value"]; ok {
		t.Errorf("set ack echoes the value: %v", setOut)
	}

	// Setting the key to the empty string deletes it.
	var delOut map[string]any
	call(t, cs, "set_memory", map[string]any{"scope": "global", "key": "diet-k", "value": ""}, &delOut)
	if delOut["key"] != "diet-k" || delOut["deleted"] != true {
		t.Errorf("delete ack: got %v, want {key:diet-k deleted:true}", delOut)
	}

	var n int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM memories WHERE key = 'diet-k' AND scope = 'global'`).Scan(&n); err != nil {
		t.Fatalf("count memories: %v", err)
	}
	if n != 0 {
		t.Errorf("key still stored after empty-value set: %d rows", n)
	}
}

func TestSetMemoryUnknownProjectErrors(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "set_memory",
		map[string]any{"scope": "project", "project": "ghost", "key": "k", "value": "v"})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}

