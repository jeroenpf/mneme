package api

import (
	"net/http"
	"strings"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// SnippetsHandler serves the read-only snippet-library REST surface.
// Writes go through the MCP save_snippet tool, not REST.
type SnippetsHandler struct{ Store store.Store }

type snippetListResponse struct {
	Items []*models.Snippet `json:"items"`
}

func (h *SnippetsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.SnippetFilter
	if p := q.Get("project"); p != "" {
		f.Project = &p
	}
	// Language is normalized to lowercase on write, so lowercase the
	// filter too — ?language=GO matches a stored "go".
	if l := strings.ToLower(q.Get("language")); l != "" {
		f.Language = &l
	}
	if tag := q.Get("tag"); tag != "" {
		f.Tags = []string{tag}
	}
	items, err := h.Store.ListSnippets(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snippetListResponse{Items: items})
}
