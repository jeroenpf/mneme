package cli

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeroenpf/mneme/internal/config"
)

// A long-lived SSE stream must not stall graceful shutdown: Shutdown waits for
// active connections to go idle, so unless request contexts are cancelled on
// shutdown, an open /api/events stream makes RunServer wait out the full
// shutdown deadline and return "context deadline exceeded" (the Ctrl-C hang).
func TestRunServerShutsDownWithOpenSSEStream(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	cfg := &config.Config{
		DSN:  "sqlite://" + filepath.Join(t.TempDir(), "shutdown.db"),
		Host: "127.0.0.1",
		Port: port,
		Env:  "test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- RunServer(ctx, cfg) }()

	base := "http://127.0.0.1:" + port
	waitForHealth(t, base, done)

	// Open an SSE stream and read its first frame so the request is
	// established server-side before we trigger shutdown.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+"/api/events", nil)
	if err != nil {
		t.Fatalf("build SSE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer resp.Body.Close()
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatalf("read SSE preamble: %v", err)
	}

	start := time.Now()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunServer returned error on shutdown: %v", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("shutdown took %v; want prompt drain of long-lived streams", elapsed)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("RunServer did not return within 15s of cancellation")
	}
}

// waitForHealth polls /health until the server accepts requests, failing fast
// if RunServer exits early (e.g. port clash or migration error).
func waitForHealth(t *testing.T, base string, done <-chan error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("RunServer exited before becoming healthy: %v", err)
		default:
		}
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server never became healthy")
}
