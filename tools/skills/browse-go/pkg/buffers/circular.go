// Package buffers provides a thread-safe circular buffer for log entries.
package buffers

import (
	"sync"
)

// Entry is a generic log entry used by the default circular buffer.
type Entry struct {
	Timestamp int64  `json:"ts"`
	Type      string `json:"type"`
	Message   string `json:"msg"`
}

// Circular is a fixed-size ring buffer that overwrites old entries.
type Circular[T any] struct {
	mu       sync.RWMutex
	entries  []T
	capacity int
	head     int
	size     int
}

// NewCircular creates a circular buffer with the given capacity.
func NewCircular[T any](capacity int) *Circular[T] {
	return &Circular[T]{
		entries:  make([]T, capacity),
		capacity: capacity,
	}
}

// Add appends an entry to the buffer.
func (c *Circular[T]) Add(e T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[c.head] = e
	c.head = (c.head + 1) % c.capacity
	if c.size < c.capacity {
		c.size++
	}
}

// All returns a copy of all entries in chronological order.
func (c *Circular[T]) All() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.size == 0 {
		return nil
	}
	out := make([]T, c.size)
	start := (c.head - c.size + c.capacity) % c.capacity
	for i := 0; i < c.size; i++ {
		out[i] = c.entries[(start+i)%c.capacity]
	}
	return out
}

// Len returns the number of entries in the buffer.
func (c *Circular[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Clear empties the buffer.
func (c *Circular[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.head = 0
	c.size = 0
	for i := range c.entries {
		var zero T
		c.entries[i] = zero
	}
}
