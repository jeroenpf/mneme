package api_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeroenpf/mneme/internal/api"
	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/live"
)

// TestEventsStream connects to GET /api/events and asserts it opens an
// SSE stream and delivers a broadcast frame. Subscribe races the request,
// so we re-Broadcast in a loop until the frame arrives.
func TestEventsStream(t *testing.T) {
	hub := live.NewHub()
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	srv := httptest.NewServer(api.Router(cfg, nil, nil, nil, nil, hub, nil))
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

// TestEventsStreamSendsInitialByte asserts the handler writes a body byte the
// instant the stream opens — before any broadcast or the 20s heartbeat. Some
// proxies (notably Vite's dev proxy) withhold the whole response until the
// first body byte; without this the browser's onopen would stall for 20s.
func TestEventsStreamSendsInitialByte(t *testing.T) {
	hub := live.NewHub()
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	srv := httptest.NewServer(api.Router(cfg, nil, nil, nil, nil, hub, nil))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	// Read the first byte with NO broadcast. Without the initial comment this
	// blocks until the 20s heartbeat and the 2s deadline trips.
	got := make(chan byte, 1)
	go func() {
		if b, err := bufio.NewReader(resp.Body).ReadByte(); err == nil {
			got <- b
		}
	}()
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("no initial byte within 2s — proxy would stall onopen until the heartbeat")
	}
}
