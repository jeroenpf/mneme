package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// flakyTransport fails the first failN round-trips with a network-style error,
// then delegates to next — used to exercise transient-failure retry.
type flakyTransport struct {
	failN int
	n     int
	next  http.RoundTripper
}

func (f *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.n++
	if f.n <= f.failN {
		return nil, fmt.Errorf("dial tcp: connection refused")
	}
	return f.next.RoundTrip(req)
}

func TestVoyageRetriesOnNetworkError(t *testing.T) {
	initialBackoff = time.Millisecond
	t.Cleanup(func() { initialBackoff = time.Second })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"embedding": []float32{0.1, 0.2}, "index": 0}},
		})
	}))
	defer srv.Close()

	c := &voyageClient{key: "k", model: "voyage-4-large", url: srv.URL,
		http: &http.Client{Transport: &flakyTransport{failN: 2, next: http.DefaultTransport}}}
	vecs, err := c.Embed(context.Background(), []string{"a"}, "document")
	if err != nil {
		t.Fatalf("Embed should retry past transient network errors: %v", err)
	}
	if len(vecs) != 1 {
		t.Fatalf("expected one vector after retries, got %d", len(vecs))
	}
}

func TestVoyageEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer auth: %q", r.Header.Get("Authorization"))
		}
		var body struct {
			Input     []string `json:"input"`
			Model     string   `json:"model"`
			InputType string   `json:"input_type"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		out := map[string]any{"data": []any{}}
		data := []any{}
		for i := range body.Input {
			data = append(data, map[string]any{"embedding": []float32{float32(i), 0.5}, "index": i})
		}
		out["data"] = data
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer srv.Close()

	c := &voyageClient{key: "test-key", model: "voyage-4-large", url: srv.URL, http: srv.Client()}
	vecs, err := c.Embed(context.Background(), []string{"a", "b"}, "document")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 2 || vecs[1][0] != 1 {
		t.Fatalf("unexpected vectors: %+v", vecs)
	}
}

func TestVoyageRetriesOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests) // first call: rate limited
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"embedding": []float32{0.1, 0.2}, "index": 0}},
		})
	}))
	defer srv.Close()

	c := &voyageClient{key: "k", model: "voyage-4-large", url: srv.URL, http: srv.Client()}
	// The impl's first backoff is 1s, so this test takes ~1s — acceptable
	// for a single retry case. (If you'd rather keep it instant, make the
	// initial backoff a package var the test can set to a millisecond.)
	vecs, err := c.Embed(context.Background(), []string{"a"}, "document")
	if err != nil {
		t.Fatalf("Embed should retry past 429: %v", err)
	}
	if calls < 2 || len(vecs) != 1 {
		t.Fatalf("expected a retry then success: calls=%d vecs=%d", calls, len(vecs))
	}
}

// TestVoyageLive hits the real API — skipped unless MNEME_VOYAGE_API_KEY is
// set. Run manually to pin the wire format; never runs in `make test`.
func TestVoyageLive(t *testing.T) {
	key := os.Getenv("MNEME_VOYAGE_API_KEY")
	if key == "" {
		t.Skip("MNEME_VOYAGE_API_KEY not set; skipping live Voyage smoke")
	}
	c := newVoyageClient(key, "voyage-4-large")
	vecs, err := c.Embed(context.Background(), []string{"zigbee coordinator swap"}, "query")
	if err != nil {
		t.Fatalf("live Embed: %v", err)
	}
	if len(vecs) != 1 || len(vecs[0]) != 1024 {
		t.Fatalf("expected one 1024-dim vector, got %d x %d", len(vecs), len(vecs[0]))
	}
}
