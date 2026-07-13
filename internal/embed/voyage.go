package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const voyageURL = "https://api.voyageai.com/v1/embeddings"

// embedDim is pinned into every request and MUST equal the embeddings
// column dimension (migration 013 → vector(1024)). Changing it requires a
// new migration.
const embedDim = 1024

type voyageClient struct {
	key   string
	model string
	url   string
	http  *http.Client
}

func newVoyageClient(key, model string) *voyageClient {
	return &voyageClient{key: key, model: model, url: voyageURL, http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *voyageClient) Model() string { return c.model }

// maxBatch is Voyage's per-request input cap (verify against docs; 128 is
// the documented example size). Larger sources are split.
const maxBatch = 128

// Embed returns one vector per input text, order-preserving. inputType is
// "document" for stored chunks and "query" for search queries. Splits into
// ≤maxBatch requests and retries each on 429/5xx with backoff, so a low
// rate-limit tier just runs slower.
func (c *voyageClient) Embed(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += maxBatch {
		end := min(start+maxBatch, len(texts))
		vecs, err := c.embedBatch(ctx, texts[start:end], inputType)
		if err != nil {
			return nil, err
		}
		out = append(out, vecs...)
	}
	return out, nil
}

func (c *voyageClient) embedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"input": texts, "model": c.model, "input_type": inputType, "output_dimension": embedDim,
	})
	backoff := time.Second
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("voyage request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("voyage call: %w", err)
		}
		// 429 (rate limit) and 5xx are retryable — back off and try again.
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			if attempt >= 5 {
				return nil, fmt.Errorf("voyage rate-limited after %d retries (status %d)", attempt, resp.StatusCode)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("voyage status %d", resp.StatusCode)
		}
		var out struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, fmt.Errorf("voyage decode: %w", err)
		}
		vecs := make([][]float32, len(out.Data))
		for _, d := range out.Data {
			if d.Index >= 0 && d.Index < len(vecs) {
				vecs[d.Index] = d.Embedding
			}
		}
		return vecs, nil
	}
}
