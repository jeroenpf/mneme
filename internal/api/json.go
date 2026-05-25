package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jeroenpfeil/mneme/internal/store"
)

// errorResponse is the wire envelope for non-2xx responses.
type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// writeStoreError translates a store-layer error into an HTTP response.
// Anything not mapped explicitly is logged and returned as 500.
func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrInvalidProject):
		writeError(w, http.StatusBadRequest, "unknown project")
	case errors.Is(err, store.ErrDuplicateID):
		writeError(w, http.StatusConflict, "duplicate id")
	default:
		slog.Error("store error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// decodeJSON pulls a JSON body into v. Returns false (and writes a 400)
// if the body is missing, malformed, or contains unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}
