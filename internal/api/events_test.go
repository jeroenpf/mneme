package api_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeroenpfeil/mneme/internal/api"
	"github.com/jeroenpfeil/mneme/internal/config"
	"github.com/jeroenpfeil/mneme/internal/live"
)

// TestEventsStream connects to GET /api/events and asserts it opens an
// SSE stream and delivers a broadcast frame. Subscribe races the request,
// so we re-Broadcast in a loop until the frame arrives.
func TestEventsStream(t *testing.T) {
	hub := live.NewHub()
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	srv := httptest.NewServer(api.Router(cfg, nil, nil, nil, nil, hub))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type %q", ct)
	}

	reader := bufio.NewReader(resp.Body)
	got := make(chan string, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "data: ") {
				got <- line
				return
			}
		}
	}()

	deadline := time.After(3 * time.Second)
	for {
		hub.Broadcast(live.Event{Type: "documents", ID: "d1"})
		select {
		case line := <-got:
			if !strings.Contains(line, `"id":"d1"`) {
				t.Fatalf("frame: %s", line)
			}
			return
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatal("no SSE frame")
		}
	}
}
