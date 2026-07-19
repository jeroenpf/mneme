package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jeroenpfeil/mneme/internal/bundle"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// BundleHandler serves the read-only context-bundle preview. The same
// assembler backs the get_context_bundle MCP tool.
type BundleHandler struct{ Store store.Store }

func (h *BundleHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	project := strings.TrimSpace(q.Get("project"))
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	var areaPtr *string
	if a := strings.TrimSpace(q.Get("area")); a != "" {
		areaPtr = &a
	}
	budget, _ := strconv.Atoi(strings.TrimSpace(q.Get("budget"))) // 0/invalid -> default
	b, err := bundle.New(h.Store).AssembleWithOptions(r.Context(), project, areaPtr, bundle.Options{TokenBudget: budget})
	if err != nil {
		writeStoreError(w, err) // ErrInvalidProject -> 400 "unknown project"
		return
	}
	writeJSON(w, http.StatusOK, b)
}
