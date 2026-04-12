package event

import "sync"

// Subscriber receives an event type and payload.
type Subscriber func(eventType string, data any)

// Emitter is a non-blocking pub/sub primitive.
type Emitter struct {
	mu      sync.RWMutex
	subs    []Subscriber
	typedOn map[string][]func(any)
}

func NewEmitter() *Emitter {
	return &Emitter{typedOn: make(map[string][]func(any))}
}

// Subscribe registers a subscriber for all events.
func (e *Emitter) Subscribe(fn Subscriber) {
	e.mu.Lock()
	e.subs = append(e.subs, fn)
	e.mu.Unlock()
}

// On registers a typed handler for a specific event type.
func (e *Emitter) On(eventType string, fn func(any)) {
	e.mu.Lock()
	e.typedOn[eventType] = append(e.typedOn[eventType], fn)
	e.mu.Unlock()
}

// Emit dispatches an event to all subscribers without blocking on slow ones.
func (e *Emitter) Emit(eventType string, data any) {
	e.mu.RLock()
	subs := make([]Subscriber, len(e.subs))
	copy(subs, e.subs)
	typed := append([]func(any){}, e.typedOn[eventType]...)
	e.mu.RUnlock()

	for _, fn := range subs {
		go safeCall(fn, eventType, data)
	}
	for _, fn := range typed {
		go safeCallTyped(fn, data)
	}
}

func safeCall(fn Subscriber, eventType string, data any) {
	defer func() { _ = recover() }()
	fn(eventType, data)
}

func safeCallTyped(fn func(any), data any) {
	defer func() { _ = recover() }()
	fn(data)
}
