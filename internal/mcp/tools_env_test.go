package mcp_test

import "testing"

func TestSetAndGetEnv(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	call(t, cs, "set_env", map[string]any{
		"project": "apollo", "key": "API_PORT", "value": "8443", "description": "https port",
	}, nil)
	call(t, cs, "set_env", map[string]any{
		"project": "apollo", "key": "DB_SERVICE", "value": "postgres",
	}, nil)

	var out struct {
		Values map[string]string `json:"values"`
	}
	call(t, cs, "get_env", map[string]any{"project": "apollo"}, &out)
	if out.Values["API_PORT"] != "8443" || out.Values["DB_SERVICE"] != "postgres" {
		t.Errorf("get_env flat map: %v", out.Values)
	}
}

func TestListEnvIncludesDescription(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")
	call(t, cs, "set_env", map[string]any{
		"project": "apollo", "key": "API_PORT", "value": "8443", "description": "https port",
	}, nil)

	var out struct {
		Items []struct {
			Key         string  `json:"key"`
			Description *string `json:"description"`
		} `json:"items"`
	}
	call(t, cs, "list_env", map[string]any{"project": "apollo"}, &out)
	if len(out.Items) != 1 || out.Items[0].Description == nil || *out.Items[0].Description != "https port" {
		t.Errorf("list_env records: %+v", out.Items)
	}
}

func TestSetEnvUnknownProjectErrors(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "set_env",
		map[string]any{"project": "ghost", "key": "k", "value": "v"})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}
