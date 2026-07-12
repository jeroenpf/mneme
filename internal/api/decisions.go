package api

import (
	"net/http"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// DecisionsHandler serves the read-only decision-log REST surface.
// Writes go through the MCP log_decision tool, not REST.
type DecisionsHandler struct{ Store store.Store }

type decisionListResponse struct {
	Items []*models.Decision `json:"items"`
}

func (h *DecisionsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.DecisionFilter
	if p := q.Get("project"); p != "" {
		f.Project = &p
	}
	if s := q.Get("status"); s != "" {
		st := models.DecisionStatus(s)
		if err := models.ValidateDecisionStatus(st); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		f.Status = &st
	}
	items, err := h.Store.ListDecisions(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decisionListResponse{Items: items})
}
