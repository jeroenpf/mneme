package api

import (
	"net/http"

	"github.com/jeroenpfeil/mneme/internal/store"
)

// SearchStatusHandler reports embedding coverage per type + whether
// embedding is enabled (a Voyage key is configured).
type SearchStatusHandler struct {
	Store   store.Store
	Enabled bool
}

type searchStatusResponse struct {
	Enabled bool                 `json:"enabled"`
	Items   []store.TypeCoverage `json:"items"`
}

func (h *SearchStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	cov, err := h.Store.EmbeddingCoverage(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, searchStatusResponse{Enabled: h.Enabled, Items: cov})
}
