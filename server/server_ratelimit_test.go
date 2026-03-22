package server

import (
	"testing"
	"time"
)

func TestIPRateLimiter_Cleanup(t *testing.T) {
	rl := newIPRateLimiter(5, 5)
	rl.allow("1.2.3.4")
	rl.allow("5.6.7.8")

	rl.mu.Lock()
	if len(rl.limiters) != 2 {
		t.Fatalf("expected 2 limiters, got %d", len(rl.limiters))
	}
	// Backdate entries to simulate old traffic
	for _, entry := range rl.limiters {
		entry.lastSeen = time.Now().Add(-2 * time.Hour)
	}
	rl.mu.Unlock()

	rl.cleanup(time.Hour)

	rl.mu.Lock()
	remaining := len(rl.limiters)
	rl.mu.Unlock()

	if remaining != 0 {
		t.Fatalf("expected 0 limiters after cleanup, got %d", remaining)
	}
}
