package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jeroenpf/mneme/internal/live"
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
	// Emit an initial comment so the very first *body* byte reaches the client
	// immediately. A header-only flush is enough for a direct connection, but
	// some proxies (notably Vite's dev proxy) withhold the response until the
	// first body byte — which would otherwise be the 20s heartbeat, stalling
	// the browser's `onopen` for 20s. This makes it fire at once.
	if _, err := w.Write([]byte(": connected\n\n")); err != nil {
		return
	}
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
			// A named `ping` event (not a `:` comment) so the browser can see
			// it: the client's liveness watchdog uses it to detect a half-open
			// connection that never fires onerror (e.g. behind a dev proxy).
			if _, err := w.Write([]byte("event: ping\ndata: {}\n\n")); err != nil {
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
