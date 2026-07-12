package mcp_test

import "testing"

func TestLogSolutionCreateAndFind(t *testing.T) {
	cs := newClient(t)
	seedProject(t, "apollo")

	var created struct {
		ID       string `json:"id"`
		Solution string `json:"solution"`
	}
	call(t, cs, "log_solution", map[string]any{
		"project":           "apollo",
		"error_description": "container startup timeout on boot",
		"solution":          "raise the healthcheck start_period",
		"tags":              []string{"docker"},
		"source_url":        "https://example.test/timeout",
	}, &created)
	if created.ID == "" {
		t.Fatal("expected a generated id")
	}

	var found struct {
		Solutions []struct {
			Solution string `json:"solution"`
		} `json:"solutions"`
	}
	call(t, cs, "find_solution", map[string]any{"query": "startup timeout"}, &found)
	if len(found.Solutions) != 1 || found.Solutions[0].Solution != "raise the healthcheck start_period" {
		t.Fatalf("find_solution: %+v", found.Solutions)
	}
}

func TestLogSolutionUnknownProject(t *testing.T) {
	cs := newClient(t)
	msg := callExpectError(t, cs, "log_solution", map[string]any{
		"project": "ghost", "error_description": "e", "solution": "s",
	})
	if msg == "" {
		t.Error("expected unknown-project error")
	}
}

func TestLogSolutionMissingFields(t *testing.T) {
	cs := newClient(t)
	// missing solution
	if msg := callExpectError(t, cs, "log_solution", map[string]any{"error_description": "e"}); msg == "" {
		t.Error("expected missing-solution error")
	}
	// missing error_description
	if msg := callExpectError(t, cs, "log_solution", map[string]any{"solution": "s"}); msg == "" {
		t.Error("expected missing-error_description error")
	}
}

func TestLogSolutionUpsert(t *testing.T) {
	cs := newClient(t)
	var created struct {
		ID string `json:"id"`
	}
	call(t, cs, "log_solution", map[string]any{
		"error_description": "flaky dns", "solution": "draft fix", "tags": []string{"dns"},
	}, &created)

	var updated struct {
		ErrorDescription string   `json:"error_description"`
		Solution         string   `json:"solution"`
		Tags             []string `json:"tags"`
	}
	call(t, cs, "log_solution", map[string]any{
		"id": created.ID, "solution": "final fix", "tags": []string{},
	}, &updated)
	if updated.Solution != "final fix" {
		t.Errorf("solution not updated: %q", updated.Solution)
	}
	if updated.ErrorDescription != "flaky dns" { // partial update preserves error_description
		t.Errorf("error_description clobbered: %q", updated.ErrorDescription)
	}
	if len(updated.Tags) != 0 { // explicit [] clears the list
		t.Errorf("tags not cleared: %+v", updated.Tags)
	}
}

func TestFindSolutionDefaultTopThree(t *testing.T) {
	cs := newClient(t)
	for i := 0; i < 4; i++ {
		call(t, cs, "log_solution", map[string]any{
			"error_description": "connection reset during deploy",
			"solution":          "retry with backoff",
		}, nil)
	}

	var found struct {
		Solutions []struct {
			Solution string `json:"solution"`
		} `json:"solutions"`
	}
	// no limit arg -> default top 3
	call(t, cs, "find_solution", map[string]any{"query": "connection reset"}, &found)
	if len(found.Solutions) != 3 {
		t.Fatalf("expected default top 3, got %d", len(found.Solutions))
	}
}
