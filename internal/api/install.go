package api

import (
	"encoding/json"
	"net/http"

	"github.com/jeroenpf/mneme/internal/appinfo"
)

// InstallHandler serves the effective install facts (URL, MCP endpoint, storage,
// embeddings state) for the local UI's Help page. The Info is computed once at
// startup — it is static for the process lifetime — and never carries secrets.
type InstallHandler struct {
	Info appinfo.Info
}

func (h *InstallHandler) Get(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.Info)
}
