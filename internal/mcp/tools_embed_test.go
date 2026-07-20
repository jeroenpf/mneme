package mcp_test

import (
	"context"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	mcpsrv "github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/store"
)

// captureEnq records the refs it is asked to enqueue.
type captureEnq struct {
	mu   sync.Mutex
	refs []embed.SourceRef
}

func (c *captureEnq) Enqueue(r embed.SourceRef) {
	c.mu.Lock()
	c.refs = append(c.refs, r)
	c.mu.Unlock()
}

func (c *captureEnq) got() []embed.SourceRef {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]embed.SourceRef(nil), c.refs...)
}

// newClientWithEnq is newClient with an injectable enqueuer, so tests can
// assert which sources a tool queues for embedding.
func newClientWithEnq(t *testing.T, enq embed.Enqueuer) *sdk.ClientSession {
	t.Helper()
	resetDB(t)
	st := store.NewWithPool(testPool)
	return connect(t, mcpsrv.New(st, enq, live.NopBroadcaster{}, nil))
}

func TestRetryFailedEmbeddings(t *testing.T) {
	enq := &captureEnq{}
	cs := newClientWithEnq(t, enq)
	ctx := context.Background()

	st := store.NewWithPool(testPool)
	if err := st.RecordEmbedFailure(ctx, "documents", "boom-doc", "kaboom"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordEmbedFailure(ctx, "decisions", "boom-dec", "kaboom"); err != nil {
		t.Fatal(err)
	}

	var out struct {
		Retried int `json:"retried"`
	}
	call(t, cs, "retry_failed_embeddings", struct{}{}, &out)
	if out.Retried != 2 {
		t.Fatalf("expected 2 sources re-enqueued, got %d", out.Retried)
	}

	got := enq.got()
	if len(got) != 2 {
		t.Fatalf("expected 2 enqueued refs, got %+v", got)
	}
	seen := map[string]bool{}
	for _, r := range got {
		seen[r.Type+"/"+r.ID] = true
	}
	if !seen["documents/boom-doc"] || !seen["decisions/boom-dec"] {
		t.Fatalf("retry did not enqueue the failed sources: %+v", got)
	}
}
