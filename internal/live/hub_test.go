package live

import "testing"

func TestHubBroadcastDelivers(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	defer cancel()
	h.Broadcast(Event{Type: "documents", ID: "d1"})
	if got := <-ch; got.Type != "documents" || got.ID != "d1" {
		t.Fatalf("got %+v", got)
	}
}

func TestHubDropsSlowConsumer(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	defer cancel()
	for i := 0; i < 20; i++ { // never blocks despite cap 16
		h.Broadcast(Event{Type: "documents", ID: "x"})
	}
	n := 0
	for len(ch) > 0 {
		<-ch
		n++
	}
	if n == 0 || n > 16 {
		t.Fatalf("buffered %d, want 1..16", n)
	}
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	cancel()
	h.Broadcast(Event{Type: "documents", ID: "d1"}) // must not panic
	if _, ok := <-ch; ok {
		t.Fatalf("channel should be closed")
	}
}
