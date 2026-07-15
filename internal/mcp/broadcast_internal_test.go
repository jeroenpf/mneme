package mcp

import (
	"sync"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/live"
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
// doc-level live event alongside the embed job — one line covers all five
// embeddable types.
func TestEnqueueBroadcasts(t *testing.T) {
	capbc := &captureBroadcaster{}
	tl := &tools{enq: embed.NopEnqueuer{}, bc: capbc}
	tl.enqueue("decisions", "d1")
	if len(capbc.events) != 1 || capbc.events[0] != (live.Event{Type: "decisions", ID: "d1"}) {
		t.Fatalf("got %+v", capbc.events)
	}
}
