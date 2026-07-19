package embed

import (
	"context"
	"sync"
	"time"
)

// rateLimiter spaces successive Wait calls by at least interval, so actual
// provider requests stay under a low RPM tier. The first call after an idle
// period passes immediately; subsequent calls are delayed to keep a steady
// 60s/rpm cadence. interval<=0 disables spacing (Wait is a no-op). Safe for
// concurrent use, though the worker calls it from a single goroutine.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time // earliest instant the next request may proceed
}

// newRateLimiter builds a limiter permitting rpm requests per minute. rpm<=0
// disables it (used in tests and on tiers without a proactive limit, where
// the client's 429 backoff is the only guard).
func newRateLimiter(rpm int) *rateLimiter {
	if rpm <= 0 {
		return &rateLimiter{}
	}
	return &rateLimiter{interval: time.Minute / time.Duration(rpm)}
}

// Wait blocks until this request's slot is due, or until ctx is cancelled.
func (l *rateLimiter) Wait(ctx context.Context) error {
	if l.interval <= 0 {
		return nil
	}
	l.mu.Lock()
	now := time.Now()
	if l.next.Before(now) {
		l.next = now
	}
	at := l.next
	l.next = l.next.Add(l.interval)
	l.mu.Unlock()

	d := time.Until(at)
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
