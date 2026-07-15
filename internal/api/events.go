package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jeroenpfeil/mneme/internal/live"
)

// EventsHandler streams live entity-change events to the browser over SSE.
// One EventSource per tab. MUST run OUTSIDE middleware.Timeout — the stream
// is deliberately long-lived.
type EventsHandler struct{ Hub *live.Hub }

const sseHeartbeat = 20 * time.Second

func (h *EventsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush() // open the stream so the client's onopen fires

	ch, cancel := h.Hub.Subscribe()
	defer cancel()
	ticker := time.NewTicker(sseHeartbeat)
	defer ticker.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case ev := <-ch:
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
