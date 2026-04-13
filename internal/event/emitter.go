package event

import (
	"log"
	"sync"
)

// Subscriber receives an event type and payload.
type Subscriber func(eventType string, data any)

// Emitter is a non-blocking pub/sub primitive.
type Emitter struct {
	mu      sync.RWMutex
	subs    []Subscriber
	typedOn map[string][]func(any)
	sem     chan struct{}
}

func NewEmitter() *Emitter {
	return &Emitter{
		typedOn: make(map[string][]func(any)),
		sem:     make(chan struct{}, 64),
	}
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
		e.dispatch(eventType, func() {
			fn(eventType, data)
		})
	}
	for _, fn := range typed {
		e.dispatch(eventType, func() {
			fn(data)
		})
	}
}

func (e *Emitter) dispatch(eventType string, fn func()) {
	select {
	case e.sem <- struct{}{}:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("event: subscriber panic for %q: %v", eventType, r)
				}
				<-e.sem
			}()
			fn()
		}()
	default:
		log.Printf("event: dropping %q notification; dispatcher saturated", eventType)
	}
}
