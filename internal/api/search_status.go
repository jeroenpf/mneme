package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jeroenpfeil/mneme/internal/store"
)

// EmbedRuntime exposes live embedding-worker state to the status endpoint and
// the manual retry control. It is nil when embeddings are disabled (no worker);
// *embed.Worker satisfies it. Kept as a local interface so the api package does
// not depend on the concrete worker for these read-only signals.
type EmbedRuntime interface {
	QueueDepth() int
	LastReconcile() time.Time
	RetryFailed(ctx context.Context) (int, error)
}

// SearchStatusHandler reports per-type embedding reconciliation buckets, the
// provider identity, the live queue depth, and the last reconciliation time —
// everything the operations UI needs to judge index health. Enabled reflects
// whether a Voyage key is configured; Model splits reconciled vs stale vectors
// and is empty when disabled; Runtime is nil when disabled.
type SearchStatusHandler struct {
	Store    store.Store
	Enabled  bool
	Provider string // "voyage" when enabled, "" (lexical-only) when disabled
	Model    string
	Runtime  EmbedRuntime
}

type providerStatus struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

type searchStatusResponse struct {
	Enabled       bool               `json:"enabled"`
	Provider      providerStatus     `json:"provider"`
	Items         []store.TypeStatus `json:"items"`
	QueueDepth    int                `json:"queue_depth"`
	LastReconcile *time.Time         `json:"last_reconcile,omitempty"`
}

func (h *SearchStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.EmbeddingStatus(r.Context(), h.Model)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	resp := searchStatusResponse{
		Enabled:  h.Enabled,
		Provider: providerStatus{Name: h.Provider, Model: h.Model, Enabled: h.Enabled},
		Items:    items,
	}
	if h.Runtime != nil {
		resp.QueueDepth = h.Runtime.QueueDepth()
		if t := h.Runtime.LastReconcile(); !t.IsZero() {
			resp.LastReconcile = &t
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// Retry re-enqueues every terminally-failed source for another embedding
// attempt (POST /api/v1/search/reindex-failed) and reports the count. With
// embeddings disabled there is nothing to retry, so it is a harmless no-op.
func (h *SearchStatusHandler) Retry(w http.ResponseWriter, r *http.Request) {
	if h.Runtime == nil {
		writeJSON(w, http.StatusOK, map[string]int{"retried": 0})
		return
	}
	n, err := h.Runtime.RetryFailed(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"retried": n})
}
