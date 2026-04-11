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

func (rb *RingBuffer[T]) Cap() int { return rb.capacity }

func (rb *RingBuffer[T]) Last() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var zero T
	if rb.count == 0 {
		return zero, false
	}
	idx := (rb.head - 1 + rb.capacity) % rb.capacity
	return rb.data[idx], true
}

func (rb *RingBuffer[T]) Reset() {
	rb.mu.Lock()
	rb.head = 0
	rb.count = 0
	rb.mu.Unlock()
}

// SlidingWindow is a count-based sliding window built on RingBuffer.
type SlidingWindow[T any] struct {
	buf *RingBuffer[T]
}

func NewSlidingWindow[T any](maxSize int) *SlidingWindow[T] {
	return &SlidingWindow[T]{buf: NewRingBuffer[T](maxSize)}
}

func (sw *SlidingWindow[T]) Add(item T) { sw.buf.Add(item) }
func (sw *SlidingWindow[T]) Slice() []T { return sw.buf.Slice() }
func (sw *SlidingWindow[T]) Len() int   { return sw.buf.Len() }
func (sw *SlidingWindow[T]) Cap() int   { return sw.buf.Cap() }
func (sw *SlidingWindow[T]) Reset()     { sw.buf.Reset() }
