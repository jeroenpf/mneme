package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// MemoryHandler serves the un-merged memory CRUD surface. The Vue memory
// page is the one sanctioned write client in the otherwise read-mostly
// UI; the global→project→area merge lives in the MCP get_memory tool.
type MemoryHandler struct{ Store store.Store }

type memoryListResponse struct {
	Items []*models.Memory `json:"items"`
}

func (h *MemoryHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.MemoryFilter
	if s := q.Get("scope"); s != "" {
		sc := models.MemoryScope(s)
		f.Scope = &sc
	}
	if p := q.Get("project"); p != "" {
		f.Project = &p
	}
	if a := q.Get("area"); a != "" {
		f.Area = &a
	}
	items, err := h.Store.ListMemory(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memoryListResponse{Items: items})
}

type memoryValueRequest struct {
	Value string `json:"value"`
}

func (h *MemoryHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	scope := models.MemoryScope(chi.URLParam(r, "scope"))
	key := chi.URLParam(r, "key")
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	area := strings.TrimSpace(r.URL.Query().Get("area"))

	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if err := models.ValidateMemoryScoping(scope, project, area); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req memoryValueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}
	m := &models.Memory{Scope: scope, Key: key, Value: req.Value}
	if project != "" {
		m.Project = &project
	}
	if area != "" {
		m.Area = &area
	}
	if err := h.Store.SetMemory(r.Context(), m); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *MemoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	scope := models.MemoryScope(chi.URLParam(r, "scope"))
	key := chi.URLParam(r, "key")
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	area := strings.TrimSpace(r.URL.Query().Get("area"))
	if err := models.ValidateMemoryScoping(scope, project, area); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var pp, ap *string
	if project != "" {
		pp = &project
	}
	if area != "" {
		ap = &area
	}
	if err := h.Store.DeleteMemory(r.Context(), scope, pp, ap, key); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
