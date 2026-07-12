package mcp_test

import "testing"

func TestGetMemoryMergesScopes(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	call(t, cs, "set_memory", map[string]any{"scope": "global", "key": "editor", "value": "neovim"}, nil)
	call(t, cs, "set_memory", map[string]any{"scope": "global", "key": "shell", "value": "zsh"}, nil)
	call(t, cs, "set_memory", map[string]any{"scope": "project", "project": "apollo", "key": "editor", "value": "goland"}, nil)

	var out struct {
		Values map[string]string `json:"values"`
	}
	call(t, cs, "get_memory", map[string]any{"scope": "project", "project": "apollo"}, &out)
	if out.Values["editor"] != "goland" { // project overrides global
		t.Errorf("editor: got %q, want goland", out.Values["editor"])
	}
	if out.Values["shell"] != "zsh" { // global still present
		t.Errorf("shell: got %q, want zsh", out.Values["shell"])
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

func TestDeleteMemoryRoundTrip(t *testing.T) {
	cs := newClient(t)
	call(t, cs, "set_memory", map[string]any{"scope": "global", "key": "k", "value": "v"}, nil)
	call(t, cs, "delete_memory", map[string]any{"scope": "global", "key": "k"}, nil)
	var out struct {
		Values map[string]string `json:"values"`
	}
	call(t, cs, "get_memory", map[string]any{"scope": "global"}, &out)
	if _, ok := out.Values["k"]; ok {
		t.Errorf("key should be gone, got %v", out.Values)
	}
}
