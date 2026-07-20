package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jeroenpf/mneme/internal/command"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// revisionSummary is the compact, body-stripped audit line the history view
// renders — the same projection the get_document_history MCP tool returns.
type revisionSummary struct {
	Revision  int       `json:"revision"`
	Op        string    `json:"op"`
	Actor     string    `json:"actor"`
	TargetIDs []string  `json:"target_ids"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type revisionsResponse struct {
	Items []revisionSummary `json:"items"`
}

// Revisions lists a document's revision history, newest first, as compact
// summaries (no body/meta). GET /documents/{id}/revisions?limit=N (limit<=0 →
// all).
func (h *DocumentsHandler) Revisions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := h.resolveDoc(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "limit must be a non-negative integer")
			return
		}
		limit = n
	}
	revs, err := h.Store.ListDocumentRevisions(r.Context(), doc.ID, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	items := make([]revisionSummary, len(revs))
	for i, rv := range revs {
		items[i] = revisionSummary{
			Revision: rv.Revision, Op: rv.Op, Actor: rv.Actor,
			TargetIDs: rv.TargetIDs, Title: rv.Title, Status: rv.Status, CreatedAt: rv.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, revisionsResponse{Items: items})
}

type restoreRequest struct {
	Revision int `json:"revision"`
}

type restoreResponse struct {
	RestoredFrom int              `json:"restored_from"`
	NewRevision  int              `json:"new_revision"`
	Doc          *models.Document `json:"doc"`
}

// Restore rewinds a document's content to a past revision by writing a new
// forward revision (history is append-only), through the shared command service
// so the restore records its own audit entry, re-embeds, and broadcasts.
// POST /documents/{id}/restore  {"revision": N}.
func (h *DocumentsHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req restoreRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Revision <= 0 {
		writeError(w, http.StatusBadRequest, "revision must be a positive integer")
		return
	}
	doc, err := h.resolveDoc(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	snap, err := h.Store.GetDocumentRevision(r.Context(), doc.ID, req.Revision)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "revision not found in this document's history")
			return
		}
		writeStoreError(w, err)
		return
	}
	// Restore the snapshot's content onto the live document; the write bumps the
	// revision and records a new "restore" audit entry (matches the MCP tool).
	doc.Title = snap.Title
	doc.Status = snap.Status
	doc.Meta = snap.Meta
	doc.Body = snap.Body
	if err := h.Writer.Update(r.Context(), doc, command.Write{
		Op:    "rest:restore",
		Actor: "rest",
		Event: live.Event{Type: "documents", ID: doc.ID, Op: "restore_document_revision"},
	}); err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("ETag", etag(doc.Revision))
	writeJSON(w, http.StatusOK, restoreResponse{
		RestoredFrom: req.Revision, NewRevision: doc.Revision, Doc: doc,
	})
}
