package embed

import (
	"context"

	"github.com/jeroenpfeil/mneme/internal/config"
)

// Client turns text into embedding vectors. A nil Client means embedding is
// disabled (no API key) — callers must check for nil and fall back to FTS.
type Client interface {
	Embed(ctx context.Context, texts []string, inputType string) ([][]float32, error)
	Model() string
}

// NewClient returns a Voyage-backed Client, or nil when no key is set.
func NewClient(cfg config.Config) Client {
	if cfg.VoyageKey == "" {
		return nil
	}
	return newVoyageClient(cfg.VoyageKey, cfg.VoyageModel)
}
