package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// EnvHandler serves the project-scoped env registry CRUD surface. Like
// MemoryHandler, the Vue env page is a sanctioned write client in the
// read-mostly UI. Every route requires a project query param — env has
// no un-scoped listing.
type EnvHandler struct{ Store store.Store }

type envListResponse struct {
	Items []*models.EnvEntry `json:"items"`
}

func (h *EnvHandler) List(w http.ResponseWriter, r *http.Request) {
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	items, err := h.Store.ListEnv(r.Context(), project)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envListResponse{Items: items})
}

type envValueRequest struct {
	Value       string  `json:"value"`
	Description *string `json:"description,omitempty"`
}

func (h *EnvHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	key := strings.TrimSpace(chi.URLParam(r, "key"))
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	var req envValueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}
	e := &models.EnvEntry{Project: project, Key: key, Value: req.Value, Description: req.Description}
	if err := h.Store.SetEnv(r.Context(), e); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *EnvHandler) Delete(w http.ResponseWriter, r *http.Request) {
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	key := strings.TrimSpace(chi.URLParam(r, "key"))
	if project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	if err := h.Store.DeleteEnv(r.Context(), project, key); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
