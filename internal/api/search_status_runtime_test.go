package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jeroenpf/mneme/internal/api"
)

// fakeRuntime satisfies api.EmbedRuntime with fixed values, so the status
// handler's runtime plumbing can be exercised without a live worker.
type fakeRuntime struct {
	depth   int
	last    time.Time
	retried int
}

func (f fakeRuntime) QueueDepth() int                          { return f.depth }
func (f fakeRuntime) LastReconcile() time.Time                 { return f.last }
func (f fakeRuntime) RetryFailed(context.Context) (int, error) { return f.retried, nil }

// The status response reports provider identity, live queue depth, and the
// last reconciliation time when a runtime is attached.
func TestSearchStatusReportsRuntimeAndProvider(t *testing.T) {
	_, st := newServer(t)
	when := time.Now().Add(-3 * time.Minute).UTC().Truncate(time.Second)
	h := &api.SearchStatusHandler{
		Store: st, Enabled: true, Provider: "voyage", Model: "voyage-4-large",
		Runtime: fakeRuntime{depth: 7, last: when},
	}

	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/v1/search/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Enabled  bool `json:"enabled"`
		Provider struct {
			Name    string `json:"name"`
			Model   string `json:"model"`
			Enabled bool   `json:"enabled"`
		} `json:"provider"`
		QueueDepth    int        `json:"queue_depth"`
		LastReconcile *time.Time `json:"last_reconcile"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Provider.Name != "voyage" || body.Provider.Model != "voyage-4-large" || !body.Provider.Enabled {
		t.Fatalf("provider wrong: %+v", body.Provider)
	}
	if body.QueueDepth != 7 {
		t.Fatalf("queue_depth = %d, want 7", body.QueueDepth)
	}
	if body.LastReconcile == nil || !body.LastReconcile.Equal(when) {
		t.Fatalf("last_reconcile = %v, want %v", body.LastReconcile, when)
	}
}

// With no runtime (embeddings disabled), the last_reconcile field is omitted
// and depth is zero — the handler must be nil-safe.
func TestSearchStatusNilRuntime(t *testing.T) {
	_, st := newServer(t)
	h := &api.SearchStatusHandler{Store: st, Enabled: false}
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/v1/search/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); !json.Valid([]byte(got)) {
		t.Fatalf("invalid json: %s", got)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if _, ok := body["last_reconcile"]; ok {
		t.Fatalf("last_reconcile should be omitted when no runtime: %v", body["last_reconcile"])
	}
}

// The retry route re-enqueues failed sources through the runtime and reports
// the count.
func TestSearchReindexFailedRoute(t *testing.T) {
	h := &api.SearchStatusHandler{Runtime: fakeRuntime{retried: 4}}
	rec := httptest.NewRecorder()
	h.Retry(rec, httptest.NewRequest(http.MethodPost, "/api/v1/search/reindex-failed", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Retried int `json:"retried"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Retried != 4 {
		t.Fatalf("retried = %d, want 4", body.Retried)
	}
}

// Retry with embeddings disabled (nil runtime) is a harmless no-op.
func TestSearchReindexFailedDisabled(t *testing.T) {
	h := &api.SearchStatusHandler{Runtime: nil}
	rec := httptest.NewRecorder()
	h.Retry(rec, httptest.NewRequest(http.MethodPost, "/api/v1/search/reindex-failed", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Retried int `json:"retried"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Retried != 0 {
		t.Fatalf("retried = %d, want 0", body.Retried)
	}
}
