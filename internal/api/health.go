package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Health struct {
	DB *pgxpool.Pool
}

func (h *Health) Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbOK := h.DB.Ping(ctx) == nil

	resp := map[string]bool{"ok": dbOK, "db": dbOK}
	w.Header().Set("Content-Type", "application/json")
	if !dbOK {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(resp)
}
