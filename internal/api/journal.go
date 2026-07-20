package api

import (
	"net/http"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// JournalHandler serves the read-only dev-journal REST surface. Writes go
// through the MCP append_journal tool, not REST.
type JournalHandler struct{ Store store.Store }

type journalListResponse struct {
	Items []*models.JournalEntry `json:"items"`
}

func (h *JournalHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.JournalFilter
	if p := q.Get("project"); p != "" {
		f.Project = &p
	}
	if s := q.Get("since"); s != "" {
		t, err := models.ParseSince(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		f.Since = &t
	}
	items, err := h.Store.ListJournalEntries(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, journalListResponse{Items: items})
}
