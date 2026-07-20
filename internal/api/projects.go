package api

import (
	"net/http"
	"strings"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/slug"
	"github.com/jeroenpf/mneme/internal/store"
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

// createProjectRequest is the wire shape for POST /projects. description
// is optional; slug is normalized to kebab-case before insertion.
type createProjectRequest struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.Slug) == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}
	p := &models.Project{
		Name:        name,
		Slug:        slug.Make(req.Slug),
		Description: normalizeDescription(req.Description),
	}
	if err := h.Store.CreateProject(r.Context(), p); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// normalizeDescription collapses a whitespace-only description to nil so
// the column stores NULL rather than an empty string.
func normalizeDescription(d *string) *string {
	if d == nil {
		return nil
	}
	if strings.TrimSpace(*d) == "" {
		return nil
	}
	return d
}
