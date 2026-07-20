package api

import (
	"net/http"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// SolutionsHandler serves the read-only error/solution REST surface. Writes
// go through the MCP log_solution tool, and ranked search through
// find_solution — REST is filter-only, not search.
type SolutionsHandler struct{ Store store.Store }

type solutionListResponse struct {
	Items []*models.Solution `json:"items"`
}

func (h *SolutionsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.SolutionFilter
	if p := q.Get("project"); p != "" {
		f.Project = &p
	}
	if tag := q.Get("tag"); tag != "" {
		f.Tags = []string{tag}
	}
	items, err := h.Store.ListSolutions(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, solutionListResponse{Items: items})
}
