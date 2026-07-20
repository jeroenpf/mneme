package mcp

import (
	"sync"
	"testing"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
)

// captureBroadcaster records every event for assertions.
type captureBroadcaster struct {
	mu     sync.Mutex
	events []live.Event
}

func (c *captureBroadcaster) Broadcast(ev live.Event) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()
}

// TestEnqueueBroadcasts proves the shared enqueue choke point emits a
// doc-level live event alongside the embed job — one line covers the four
// non-document embeddable types.
func TestEnqueueBroadcasts(t *testing.T) {
	capbc := &captureBroadcaster{}
	tl := &tools{enq: embed.NopEnqueuer{}, bc: capbc}
	tl.enqueue("decisions", "d1")
	if len(capbc.events) != 1 || capbc.events[0] != (live.Event{Type: "decisions", ID: "d1"}) {
		t.Fatalf("got %+v", capbc.events)
	}
}

// TestEnqueueSkipsDocumentsBroadcast pins the P3 split: documents are
// excluded from enqueue's generic broadcast because their edit handlers
// broadcast themselves with a block id — a generic doc-level event here
// would duplicate that. Embedding is still enqueued (asserted elsewhere).
func TestEnqueueSkipsDocumentsBroadcast(t *testing.T) {
	capbc := &captureBroadcaster{}
	tl := &tools{enq: embed.NopEnqueuer{}, bc: capbc}
	tl.enqueue("documents", "d1")
	if len(capbc.events) != 0 {
		t.Fatalf("enqueue must not broadcast documents (handlers do, with a block id); got %+v", capbc.events)
	}
}
