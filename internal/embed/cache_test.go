package embed

import (
	"context"
	"errors"
	"testing"
)

// countingClient counts Embed calls so cache hits/misses are observable.
type countingClient struct {
	calls int
	err   error
}

func (c *countingClient) Model() string { return "fake" }
func (c *countingClient) Embed(_ context.Context, texts []string, _ string) ([][]float32, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	out := make([][]float32, len(texts))
	for i, tx := range texts {
		out[i] = []float32{float32(len(tx))}
	}
	return out, nil
}

func TestCachingClientCachesRepeatQueries(t *testing.T) {
	under := &countingClient{}
	c := newCachingClient(under, 8)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := c.Embed(ctx, []string{"zigbee coordinator"}, "query"); err != nil {
			t.Fatal(err)
		}
	}
	if under.calls != 1 {
		t.Fatalf("repeat query embeds should hit the cache, underlying calls=%d", under.calls)
	}
}

func TestCachingClientDoesNotCacheDocuments(t *testing.T) {
	under := &countingClient{}
	c := newCachingClient(under, 8)
	ctx := context.Background()

	// Document embeds (the worker's path) must always pass through.
	_, _ = c.Embed(ctx, []string{"chunk"}, "document")
	_, _ = c.Embed(ctx, []string{"chunk"}, "document")
	if under.calls != 2 {
		t.Fatalf("document embeds must not be cached, calls=%d", under.calls)
	}
}

func TestCachingClientDistinctQueriesMiss(t *testing.T) {
	under := &countingClient{}
	c := newCachingClient(under, 8)
	ctx := context.Background()
	_, _ = c.Embed(ctx, []string{"a"}, "query")
	_, _ = c.Embed(ctx, []string{"b"}, "query")
	if under.calls != 2 {
		t.Fatalf("distinct queries should both call through, calls=%d", under.calls)
	}
}

func TestCachingClientDoesNotCacheErrors(t *testing.T) {
	under := &countingClient{err: errors.New("boom")}
	c := newCachingClient(under, 8)
	ctx := context.Background()
	if _, err := c.Embed(ctx, []string{"q"}, "query"); err == nil {
		t.Fatal("expected the underlying error to surface")
	}
	under.err = nil
	if _, err := c.Embed(ctx, []string{"q"}, "query"); err != nil {
		t.Fatal(err)
	}
	if under.calls != 2 {
		t.Fatalf("a failed embed must not be cached, calls=%d", under.calls)
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	l := newLRU(2)
	l.put("a", []float32{1})
	l.put("b", []float32{2})
	l.put("c", []float32{3}) // evicts "a" (oldest)
	if _, ok := l.get("a"); ok {
		t.Fatal("a should have been evicted")
	}
	if _, ok := l.get("b"); !ok {
		t.Fatal("b should remain")
	}
	if _, ok := l.get("c"); !ok {
		t.Fatal("c should remain")
	}
}

func TestLRURefreshesOnGet(t *testing.T) {
	l := newLRU(2)
	l.put("a", []float32{1})
	l.put("b", []float32{2})
	l.get("a")             // refresh a → b is now least-recently-used
	l.put("c", []float32{3}) // evicts b
	if _, ok := l.get("b"); ok {
		t.Fatal("b should have been evicted after a was refreshed")
	}
	if _, ok := l.get("a"); !ok {
		t.Fatal("a should remain")
	}
}
