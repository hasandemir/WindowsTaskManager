package ai

import (
	"sync"
	"time"
)

// TokenBucket is a minimal per-minute rate limiter.
type TokenBucket struct {
	mu       sync.Mutex
	max      int
	tokens   int
	lastFill time.Time
}

func NewTokenBucket(perMinute int) *TokenBucket {
	if perMinute < 1 {
		perMinute = 1
	}
	return &TokenBucket{
		max:      perMinute,
		tokens:   perMinute,
		lastFill: time.Now(),
	}
}

// Take consumes a token if available; returns false if not.
func (tb *TokenBucket) Take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	if tb.tokens <= 0 {
		return false
	}
	tb.tokens--
	return true
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastFill)
	if elapsed <= 0 {
		return
	}
	add := int(float64(tb.max) * elapsed.Minutes())
	if add > 0 {
		tb.tokens += add
		if tb.tokens > tb.max {
			tb.tokens = tb.max
		}
		tb.lastFill = now
	}
}

// Available reports current bucket level.
func (tb *TokenBucket) Available() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}
