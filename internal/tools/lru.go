// Package tools provides agent tools for CDD.
package tools

import (
	"sync"
)

// LRUCache is a generic thread-safe LRU cache with O(1) operations.
// It uses a doubly-linked list for LRU ordering and a map for O(1) lookups.
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*lruNode[K, V]
	head     *lruNode[K, V] // Most recently used
	tail     *lruNode[K, V] // Least recently used
	mu       sync.RWMutex

	// Metrics
	hits   int64
	misses int64
}

// lruNode is a node in the doubly-linked list.
type lruNode[K comparable, V any] struct {
	key   K
	value V
	prev  *lruNode[K, V]
	next  *lruNode[K, V]
}

// NewLRUCache creates a new LRU cache with the given capacity.
// Capacity must be at least 1.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity < 1 {
		capacity = 1
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*lruNode[K, V]),
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if found, zero value and false otherwise.
// Accessing a key moves it to the front (most recently used).
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.items[key]
	if !exists {
		c.misses++
		var zero V
		return zero, false
	}

	c.hits++
	c.moveToFront(node)
	return node.value, true
}

// Put adds or updates a value in the cache.
// If the cache is at capacity, the least recently used item is evicted.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing
	if node, exists := c.items[key]; exists {
		node.value = value
		c.moveToFront(node)
		return
	}

	// Create new node
	node := &lruNode[K, V]{key: key, value: value}
	c.items[key] = node
	c.addToFront(node)

	// Evict if over capacity
	if len(c.items) > c.capacity {
		c.removeTail()
	}
}

// Delete removes a key from the cache.
// Returns true if the key was present.
func (c *LRUCache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.items[key]
	if !exists {
		return false
	}

	c.removeNode(node)
	delete(c.items, key)
	return true
}

// Len returns the current number of items in the cache.
func (c *LRUCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*lruNode[K, V])
	c.head = nil
	c.tail = nil
}

// Metrics returns cache hit/miss statistics.
func (c *LRUCache[K, V]) Metrics() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

// HitRate returns the cache hit rate as a percentage (0-100).
// Returns 0 if no requests have been made.
func (c *LRUCache[K, V]) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	return float64(c.hits) / float64(total) * 100
}

// moveToFront moves an existing node to the front of the list.
func (c *LRUCache[K, V]) moveToFront(node *lruNode[K, V]) {
	if node == c.head {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// addToFront adds a node to the front of the list.
func (c *LRUCache[K, V]) addToFront(node *lruNode[K, V]) {
	node.prev = nil
	node.next = c.head

	if c.head != nil {
		c.head.prev = node
	}
	c.head = node

	if c.tail == nil {
		c.tail = node
	}
}

// removeNode removes a node from the list.
func (c *LRUCache[K, V]) removeNode(node *lruNode[K, V]) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
}

// removeTail removes and returns the least recently used node.
func (c *LRUCache[K, V]) removeTail() {
	if c.tail == nil {
		return
	}

	delete(c.items, c.tail.key)

	if c.tail.prev != nil {
		c.tail.prev.next = nil
		c.tail = c.tail.prev
	} else {
		c.head = nil
		c.tail = nil
	}
}
