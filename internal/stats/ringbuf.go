package stats

import "sync"

// RingBuffer is a fixed-capacity circular buffer with concurrent reads.
type RingBuffer[T any] struct {
	mu       sync.RWMutex
	data     []T
	capacity int
	head     int
	count    int
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer[T]{
		data:     make([]T, capacity),
		capacity: capacity,
	}
}

func (rb *RingBuffer[T]) Add(item T) {
	rb.mu.Lock()
	rb.data[rb.head] = item
	rb.head = (rb.head + 1) % rb.capacity
	if rb.count < rb.capacity {
		rb.count++
	}
	rb.mu.Unlock()
}

func (rb *RingBuffer[T]) Slice() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	result := make([]T, rb.count)
	start := (rb.head - rb.count + rb.capacity) % rb.capacity
	for i := 0; i < rb.count; i++ {
		result[i] = rb.data[(start+i)%rb.capacity]
	}
	return result
}

func (rb *RingBuffer[T]) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
