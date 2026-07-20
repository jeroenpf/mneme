package mcp

import (
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
)

func TestFlattenEnv(t *testing.T) {
	got := flattenEnv([]*models.EnvEntry{
		{Key: "A", Value: "1"},
		{Key: "B", Value: "2"},
	})
	if len(got) != 2 || got["A"] != "1" || got["B"] != "2" {
		t.Fatalf("flattenEnv: %v", got)
	}
}
