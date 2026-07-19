package api

import (
	"net/http"

	"github.com/jeroenpfeil/mneme/internal/store"
)

// SearchStatusHandler reports per-type embedding reconciliation buckets +
// whether embedding is enabled (a Voyage key is configured). Model is the
// current embedding model, used to split reconciled vs stale vectors; it is
// empty when embedding is disabled.
type SearchStatusHandler struct {
	Store   store.Store
	Enabled bool
	Model   string
}

type searchStatusResponse struct {
	Enabled bool               `json:"enabled"`
	Items   []store.TypeStatus `json:"items"`
}

func (h *SearchStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.EmbeddingStatus(r.Context(), h.Model)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, searchStatusResponse{Enabled: h.Enabled, Items: items})
}
