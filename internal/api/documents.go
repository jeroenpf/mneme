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

	"github.com/jeroenpfeil/mneme/internal/docmeta"
	"github.com/jeroenpfeil/mneme/internal/ids"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/slug"
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

	doc, err := docmeta.FromMeta(req.Meta, req.Body)
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
	base := slug.Make(doc.Title)
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
	// Accept a doc_ public id too, so a pasted mneme://document/doc_… reference
	// opens /doc/doc_… without the UI having to map it back to the slug first.
	if errors.Is(err, store.ErrNotFound) && ids.ValidFor(ids.KindDocument, id) {
		doc, err = h.Store.GetDocumentByPublicID(r.Context(), id)
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	// Expose the revision as an ETag so a conditional PATCH (If-Match) can
	// detect a concurrent edit (roadmap P6).
	w.Header().Set("ETag", etag(doc.Revision))
	writeJSON(w, http.StatusOK, doc)
}

// etag renders a document revision as a (weak-free) HTTP entity tag.
func etag(revision int) string { return fmt.Sprintf("%q", strconv.Itoa(revision)) }

// parseIfMatch extracts an expected revision from an If-Match header
// (e.g. `"3"`). Returns nil when the header is absent or unparseable, so a
// caller that doesn't send one keeps last-writer-wins semantics.
func parseIfMatch(r *http.Request) *int {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" || raw == "*" {
		return nil
	}
	raw = strings.Trim(raw, `"`)
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &n
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
		merged, err := docmeta.FromMeta(m, doc.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		docmeta.ApplyTo(doc, merged)
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

	if err := h.Store.UpdateDocument(r.Context(), doc, parseIfMatch(r)); err != nil {
		var conflict *store.RevisionConflictError
		if errors.As(err, &conflict) {
			w.Header().Set("ETag", etag(conflict.Current))
			writeError(w, http.StatusPreconditionFailed, fmt.Sprintf("revision conflict: document is at revision %d, re-read before retrying", conflict.Current))
			return
		}
		writeStoreError(w, err)
		return
	}
	w.Header().Set("ETag", etag(doc.Revision))
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
