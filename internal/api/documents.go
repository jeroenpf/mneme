package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
	maxSlugAttempts  = 100
)

// DocumentsHandler holds the dependencies the document endpoints need.
type DocumentsHandler struct {
	Store store.Store
}

// documentListResponse is the wire shape for GET /documents.
type documentListResponse struct {
	Items      []*models.Document `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

func (h *DocumentsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := defaultListLimit
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if n > maxListLimit {
			n = maxListLimit
		}
		limit = n
	}

	offset := 0
	if raw := q.Get("cursor"); raw != "" {
		n, err := decodeCursor(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		offset = n
	}

	filter := store.Filter{
		Limit:  limit + 1, // fetch one extra to determine next_cursor
		Offset: offset,
	}
	if v := q.Get("project"); v != "" {
		filter.Project = &v
	}
	if v := q.Get("type"); v != "" {
		filter.Type = &v
	}
	if v := q.Get("status"); v != "" {
		filter.Status = &v
	}
	if v := q.Get("tags"); v != "" {
		filter.Tags = splitCSV(v)
	}

	var (
		docs []*models.Document
		err  error
	)
	if search := q.Get("q"); search != "" {
		docs, err = h.Store.SearchDocuments(r.Context(), search, filter)
	} else {
		docs, err = h.Store.ListDocuments(r.Context(), filter)
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}

	resp := documentListResponse{Items: docs}
	if len(docs) > limit {
		resp.Items = docs[:limit]
		next := encodeCursor(offset + limit)
		resp.NextCursor = &next
	}
	writeJSON(w, http.StatusOK, resp)
}

// createDocumentRequest is the wire shape for POST /documents.
// meta and body are pass-through JSONB objects; the handler also
// promotes known meta fields (title, type, project, ...) into typed
// Document columns.
type createDocumentRequest struct {
	Meta map[string]any `json:"meta"`
	Body map[string]any `json:"body"`
}

func (h *DocumentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDocumentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Meta == nil {
		writeError(w, http.StatusBadRequest, "meta is required")
		return
	}
	if req.Body == nil {
		req.Body = map[string]any{}
	}

	doc, err := docFromMeta(req.Meta, req.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if doc.Title == "" {
		writeError(w, http.StatusBadRequest, "meta.title is required")
		return
	}
	if doc.Type == "" {
		writeError(w, http.StatusBadRequest, "meta.type is required")
		return
	}

	if err := h.createWithSlug(r.Context(), doc); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// createWithSlug derives the document ID from the title and retries on
// PK conflict (base, base-2, base-3, ...) up to maxSlugAttempts.
func (h *DocumentsHandler) createWithSlug(ctx context.Context, doc *models.Document) error {
	base := slugify(doc.Title)
	for i := 0; i < maxSlugAttempts; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		doc.ID = candidate
		err := h.Store.CreateDocument(ctx, doc)
		if err == nil {
			return nil
		}
		if errors.Is(err, store.ErrDuplicateID) {
			continue
		}
		return err
	}
	return fmt.Errorf("could not allocate slug for %q after %d attempts", base, maxSlugAttempts)
}

func (h *DocumentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := h.Store.GetDocument(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// updateDocumentRequest is the wire shape for PATCH /documents/:id.
// Each field is optional; omitted fields are left untouched. Pointers
// to maps would have been ambiguous with JSON null, so we use
// json.RawMessage and check len > 0.
type updateDocumentRequest struct {
	Meta         json.RawMessage `json:"meta,omitempty"`
	Body         json.RawMessage `json:"body,omitempty"`
	Status       *string         `json:"status,omitempty"`
	PhaseCurrent *int            `json:"phase_current,omitempty"`
}

func (h *DocumentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateDocumentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	doc, err := h.Store.GetDocument(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	if len(req.Meta) > 0 {
		var m map[string]any
		if err := json.Unmarshal(req.Meta, &m); err != nil {
			writeError(w, http.StatusBadRequest, "meta is not a JSON object")
			return
		}
		// Promote known meta fields back onto typed columns so filters
		// and CHECK constraints keep working after a partial update.
		merged, err := docFromMeta(m, doc.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		applyMetaToDocument(doc, merged)
	}
	if len(req.Body) > 0 {
		var b map[string]any
		if err := json.Unmarshal(req.Body, &b); err != nil {
			writeError(w, http.StatusBadRequest, "body is not a JSON object")
			return
		}
		doc.Body = b
	}
	if req.Status != nil {
		doc.Status = *req.Status
	}
	if req.PhaseCurrent != nil {
		doc.PhaseCurrent = req.PhaseCurrent
	}

	if err := h.Store.UpdateDocument(r.Context(), doc); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DocumentsHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.ArchiveDocument(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// docFromMeta builds a Document by promoting known top-level meta keys
// onto typed columns. Unknown keys remain on doc.Meta. Returns an error
// when a known key has the wrong type.
func docFromMeta(meta, body map[string]any) (*models.Document, error) {
	d := &models.Document{
		Status: models.StatusTodo,
		Tags:   []string{},
		Meta:   map[string]any{},
		Body:   body,
	}

	// Known column-mapped meta keys. Unknown keys flow through to Meta.
	for k, v := range meta {
		switch k {
		case "title":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.title must be a string")
			}
			d.Title = s
		case "type":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.type must be a string")
			}
			d.Type = s
		case "project":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.project must be a string")
			}
			d.Project = &s
		case "category":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.category must be a string")
			}
			d.Category = &s
		case "status":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.status must be a string")
			}
			d.Status = s
		case "ticket":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.ticket must be a string")
			}
			d.Ticket = &s
		case "repo":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("meta.repo must be a string")
			}
			d.Repo = &s
		case "tags":
			tags, err := castStringSlice(v)
			if err != nil {
				return nil, fmt.Errorf("meta.tags: %w", err)
			}
			d.Tags = tags
		case "phase_current":
			n, err := castInt(v)
			if err != nil {
				return nil, fmt.Errorf("meta.phase_current: %w", err)
			}
			d.PhaseCurrent = &n
			// Also keep raw value in Meta so the original object round-trips.
			d.Meta[k] = v
		case "phase_total":
			n, err := castInt(v)
			if err != nil {
				return nil, fmt.Errorf("meta.phase_total: %w", err)
			}
			d.PhaseTotal = &n
			d.Meta[k] = v
		default:
			d.Meta[k] = v
		}
	}
	return d, nil
}

// applyMetaToDocument copies the typed columns + Meta from src onto
// dst, leaving Body, ID, timestamps untouched. Used by PATCH.
func applyMetaToDocument(dst, src *models.Document) {
	dst.Title = src.Title
	dst.Project = src.Project
	dst.Category = src.Category
	dst.Type = src.Type
	dst.Status = src.Status
	dst.Ticket = src.Ticket
	dst.Repo = src.Repo
	dst.Tags = src.Tags
	dst.PhaseCurrent = src.PhaseCurrent
	dst.PhaseTotal = src.PhaseTotal
	dst.Meta = src.Meta
}

func castStringSlice(v any) ([]string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, errors.New("must be an array of strings")
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, errors.New("must be an array of strings")
		}
		out = append(out, s)
	}
	return out, nil
}

func castInt(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	}
	return 0, errors.New("must be an integer")
}

func encodeCursor(offset int) string {
	return base64.URLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(s string) (int, error) {
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(string(raw))
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errors.New("negative offset")
	}
	return n, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
