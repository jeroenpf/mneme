package api

import (
	"net/http"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

type ProjectsHandler struct {
	Store store.Store
}

type projectListResponse struct {
	Items []*models.ProjectStats `json:"items"`
}

func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.ListProjects(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectListResponse{Items: items})
}
