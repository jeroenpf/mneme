package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// SearchHandler serves the read-only unified search. The same store.Search
// backs the MCP `search` tool. Client (may be nil) embeds the query for
// hybrid ranking; nil ⇒ FTS-only.
type SearchHandler struct {
	Store  store.Store
	Client embed.Client
}

type searchListResponse struct {
	Items []*models.SearchHit `json:"items"`
}

func (h *SearchHandler) Get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}
	f := store.SearchFilter{}
	if ts := strings.TrimSpace(q.Get("types")); ts != "" {
		for _, t := range strings.Split(ts, ",") {
			if t = strings.TrimSpace(t); t != "" {
				f.Types = append(f.Types, t)
			}
		}
	}
	if p := strings.TrimSpace(q.Get("project")); p != "" {
		f.Project = &p
	}
	if n, err := strconv.Atoi(strings.TrimSpace(q.Get("limit"))); err == nil && n > 0 {
		f.Limit = n
	}
	// Embed the query for hybrid ranking when a client is configured. Any
	// Voyage error degrades to FTS-only (Vector stays nil) — search never
	// hard-fails because embeddings are down.
	if h.Client != nil {
		if vecs, err := h.Client.Embed(r.Context(), []string{query}, "query"); err == nil && len(vecs) == 1 {
			f.Vector = vecs[0]
		} else if err != nil {
			slog.Warn("search query embed failed; falling back to FTS-only", "err", err)
		}
	}
	hits, err := h.Store.Search(r.Context(), query, f)
	if err != nil {
		if errors.Is(err, store.ErrInvalidSearchType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, searchListResponse{Items: hits})
}
