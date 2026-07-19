package embed

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiterSpacesRequests(t *testing.T) {
	const interval = 40 * time.Millisecond
	l := &rateLimiter{interval: interval}
	ctx := context.Background()

	// First request goes immediately; the next two are each spaced by
	// ~interval, so three requests take at least 2*interval.
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed < 2*interval-10*time.Millisecond {
		t.Fatalf("three requests should span >= 2*interval (%v), took %v", 2*interval, elapsed)
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	l := newRateLimiter(0) // rpm<=0 disables spacing
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := l.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("disabled limiter must not sleep, took %v", elapsed)
	}
}

func TestRateLimiterRespectsContext(t *testing.T) {
	l := &rateLimiter{interval: time.Hour}
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("first Wait should pass immediately: %v", err)
	}
	// The second Wait would block ~1h; a cancelled ctx must abort it.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Wait(ctx); err == nil {
		t.Fatal("Wait must return the context error when cancelled")
	}
}
