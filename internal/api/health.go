package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger is the minimal contract Health needs. Implemented by *store.PostgresStore.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Health struct {
	Store Pinger
}

func (h *Health) Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbOK := h.Store.Ping(ctx) == nil

	resp := map[string]bool{"ok": dbOK, "db": dbOK}
	w.Header().Set("Content-Type", "application/json")
	if !dbOK {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
