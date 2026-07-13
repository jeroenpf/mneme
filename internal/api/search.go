package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// SearchHandler serves the read-only unified search. The same store.Search
// backs the MCP `search` tool.
type SearchHandler struct{ Store store.Store }

type searchListResponse struct {
	Items []*models.SearchHit `json:"items"`
}

func (h *SearchHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}
	f := store.SearchFilter{}
	if ts := strings.TrimSpace(q.Get("types")); ts != "" {
		for _, t := range strings.Split(ts, ",") {
			if t = strings.TrimSpace(t); t != "" {
				f.Types = append(f.Types, t)
			}
		}
	}
	if p := strings.TrimSpace(q.Get("project")); p != "" {
		f.Project = &p
	}
	if n, err := strconv.Atoi(strings.TrimSpace(q.Get("limit"))); err == nil && n > 0 {
		f.Limit = n
	}
	hits, err := h.Store.Search(r.Context(), query, f)
	if err != nil {
		if errors.Is(err, store.ErrInvalidSearchType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, searchListResponse{Items: hits})
}
