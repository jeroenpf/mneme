package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpfeil/mneme/internal/embed"
)

// retryFailedInput takes no arguments — it retries every recorded failure.
type retryFailedInput struct{}

type retryFailedOutput struct {
	Retried int `json:"retried"`
}

// retryFailedEmbeddings re-enqueues every source with a recorded terminal
// embed failure (the "failed" count in search status). The worker clears each
// failure as it succeeds; sources since deleted are purged on reprocessing.
func (t *tools) retryFailedEmbeddings(ctx context.Context, _ *sdk.CallToolRequest, _ retryFailedInput) (*sdk.CallToolResult, *retryFailedOutput, error) {
	refs, err := t.store.FailedSourceRefs(ctx)
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	for _, r := range refs {
		t.enq.Enqueue(embed.SourceRef{Type: r.Type, ID: r.ID})
	}
	return nil, &retryFailedOutput{Retried: len(refs)}, nil
}
