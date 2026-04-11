package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// cacheEntry holds a single cached LLM response.
type cacheEntry struct {
	answer    string
	createdAt time.Time
}

// Cache is a small in-memory TTL cache keyed by prompt hash.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	maxSize int
}

func NewCache(ttl time.Duration, maxSize int) *Cache {
	if maxSize < 16 {
		maxSize = 16
	}
	return &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

func keyOf(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

// Get returns the cached answer for `prompt` if present and unexpired.
func (c *Cache) Get(prompt string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[keyOf(prompt)]
	if !ok {
		return "", false
	}
	if time.Since(e.createdAt) > c.ttl {
		return "", false
	}
	return e.answer, true
}

// Set stores an answer in the cache, evicting oldest when at capacity.
func (c *Cache) Set(prompt, answer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		var oldestKey string
		var oldestTime time.Time
		first := true
		for k, e := range c.entries {
			if first || e.createdAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.createdAt
				first = false
			}
		}
		delete(c.entries, oldestKey)
	}
	c.entries[keyOf(prompt)] = cacheEntry{answer: answer, createdAt: time.Now()}
}

// Size returns current cache occupancy.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
