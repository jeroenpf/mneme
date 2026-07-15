// Package live is a tiny in-process pub/sub hub for pushing entity-change
// events to connected SSE clients. Transport-agnostic: it knows nothing
// about HTTP. One hub is shared by the MCP write path (Broadcast after a
// successful write) and the /api/events handler (Subscribe per browser).
package live

import "sync"

// Event is one entity-change notification. Type is the entity kind and
// matches the enqueue sourceType vocabulary (documents, decisions,
// snippets, solutions, journal, memory, env). ID identifies the changed
// entity and doubles as a list-row flash target. Project scopes
// project-specific views (env). BlockID/Op are optional hints (P3).
type Event struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Project string `json:"project,omitempty"`
	BlockID string `json:"blockId,omitempty"`
	Op      string `json:"op,omitempty"`
}

// Broadcaster is the write-side dependency; MCP tools depend on it so they
// stay agnostic to whether live updates are wired.
type Broadcaster interface{ Broadcast(Event) }

// NopBroadcaster is installed when no hub is wired (tests).
type NopBroadcaster struct{}

func (NopBroadcaster) Broadcast(Event) {}

// Hub fans Broadcast out to every current subscriber. Safe for concurrent use.
type Hub struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

func NewHub() *Hub { return &Hub{subs: map[chan Event]struct{}{}} }

// Subscribe registers a subscriber, returning its channel plus a cancel
// that unregisters and closes it (idempotent).
func (h *Hub) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, ch)
			h.mu.Unlock()
			close(ch)
		})
	}
}

// Broadcast delivers ev to every subscriber without blocking: a full buffer
// is skipped (that browser reconnects and resyncs), so a slow client never
// stalls an MCP write. Fire-and-forget by design.
func (h *Hub) Broadcast(ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
