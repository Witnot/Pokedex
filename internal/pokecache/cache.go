package pokecache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	mu       sync.Mutex
	entries  map[string]cacheEntry
	interval time.Duration
}

// NewCache creates a new cache and starts the reapLoop.
func NewCache(interval time.Duration) *Cache {
	c := &Cache{
		entries:  make(map[string]cacheEntry),
		interval: interval,
	}

	// Start background reaper
	go c.reapLoop()

	return c
}

// Add inserts a new entry into the cache.
func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		createdAt: time.Now(),
		val:       val,
	}
}

// Get retrieves an entry from the cache.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return entry.val, true
}

// reapLoop periodically removes expired entries.
func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for k, v := range c.entries {
			if time.Since(v.createdAt) > c.interval {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
