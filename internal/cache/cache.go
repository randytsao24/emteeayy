// Package cache provides a generic TTL cache
package cache

import (
	"sync"
	"time"
)

// item wraps a cached value with its expiration time
type item[T any] struct {
	value     T
	expiresAt time.Time
}

// Cache is a generic thread-safe cache with TTL expiration
type Cache[T any] struct {
	items map[string]item[T]
	mu    sync.RWMutex
	ttl   time.Duration
	stop  chan struct{}
}

// New creates a cache with the specified TTL
func New[T any](ttl time.Duration) *Cache[T] {
	c := &Cache[T]{
		items: make(map[string]item[T]),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// Get retrieves a value, returning (value, true) if found and not expired
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		var zero T
		return zero, false
	}
	return item.value, true
}

// Set stores a value with the cache's TTL
func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = item[T]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a key from the cache
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]item[T])
}

// Size returns the number of items (including expired)
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Close stops the background cleanup goroutine
func (c *Cache[T]) Close() {
	close(c.stop)
}

// cleanup runs periodically to remove expired items
func (c *Cache[T]) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stop:
			return
		}
	}
}

func (c *Cache[T]) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}
