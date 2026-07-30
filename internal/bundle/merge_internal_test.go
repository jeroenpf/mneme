package bundle

import (
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
)

func TestMergeMemoryPrecedence(t *testing.T) {
	global := []*models.Memory{{Key: "a", Value: "g"}, {Key: "b", Value: "g"}}
	project := []*models.Memory{{Key: "b", Value: "p"}, {Key: "c", Value: "p"}}
	area := []*models.Memory{{Key: "c", Value: "a"}}
	got := mergeMemory(global, project, area)
	want := map[string]string{"a": "g", "b": "p", "c": "a"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q: got %q, want %q", k, got[k], v)
		}
	}
}
