package embed

import (
	"container/list"
	"context"
	"sync"
)

// lruCache is a small bounded string→vector cache with least-recently-used
// eviction. Safe for concurrent use.
type lruCache struct {
	mu  sync.Mutex
	cap int
	ll  *list.List // front = most-recently-used
	idx map[string]*list.Element
}

type lruEntry struct {
	key string
	val []float32
}

func newLRU(capacity int) *lruCache {
	if capacity < 1 {
		capacity = 1
	}
	return &lruCache{cap: capacity, ll: list.New(), idx: make(map[string]*list.Element, capacity)}
}

func (c *lruCache) get(key string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.idx[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*lruEntry).val, true
	}
	return nil, false
}

func (c *lruCache) put(key string, val []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.idx[key]; ok {
		el.Value.(*lruEntry).val = val
		c.ll.MoveToFront(el)
		return
	}
	c.idx[key] = c.ll.PushFront(&lruEntry{key: key, val: val})
	if c.ll.Len() > c.cap {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.idx, oldest.Value.(*lruEntry).key)
		}
	}
}

// cachingClient decorates a Client with a bounded cache for query embeddings.
// Only single-text "query" embeds are cached — document embeds run once per
// chunk, so caching them would only waste memory. Document calls pass through.
type cachingClient struct {
	Client
	cache *lruCache
}

// NewCachingClient wraps c so repeated search-query embeds reuse a cached
// vector instead of re-calling the provider. capacity bounds the cache.
func NewCachingClient(c Client, capacity int) Client {
	return &cachingClient{Client: c, cache: newLRU(capacity)}
}

// newCachingClient is the unexported constructor used in tests.
func newCachingClient(c Client, capacity int) *cachingClient {
	return &cachingClient{Client: c, cache: newLRU(capacity)}
}

func (c *cachingClient) Embed(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if inputType != "query" || len(texts) != 1 {
		return c.Client.Embed(ctx, texts, inputType)
	}
	if v, ok := c.cache.get(texts[0]); ok {
		return [][]float32{v}, nil
	}
	vecs, err := c.Client.Embed(ctx, texts, inputType)
	if err != nil {
		return nil, err
	}
	if len(vecs) == 1 {
		c.cache.put(texts[0], vecs[0])
	}
	return vecs, nil
}
