package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/event"
)

// sseClient represents one connected stream subscriber.
type sseClient struct {
	id     uint64
	send   chan sseMsg
	closed chan struct{}
}

type sseMsg struct {
	event string
	data  []byte
}

// SSEHub fans out emitter events to all connected SSE clients. It throttles
// per-event-type to avoid overwhelming slow clients.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[uint64]*sseClient
	nextID  uint64
}

func NewSSEHub(emitter *event.Emitter) *SSEHub {
	hub := &SSEHub{clients: make(map[uint64]*sseClient)}
	emitter.Subscribe(func(eventType string, data any) {
		hub.broadcast(eventType, data)
	})
	return hub
}

func (h *SSEHub) broadcast(eventType string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := sseMsg{event: eventType, data: payload}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// Client is slow; drop the event for this client.
		}
	}
}

// Handler returns the SSE HTTP handler.
func (h *SSEHub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		client := &sseClient{
			send:   make(chan sseMsg, 64),
			closed: make(chan struct{}),
		}
		h.mu.Lock()
		h.nextID++
		client.id = h.nextID
		h.clients[client.id] = client
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.clients, client.id)
			h.mu.Unlock()
			close(client.closed)
		}()

		// Initial hello + heartbeat ticker.
		fmt.Fprintf(w, "event: hello\ndata: {\"client\":%d}\n\n", client.id)
		flusher.Flush()

		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				fmt.Fprintf(w, ": ping %d\n\n", time.Now().Unix())
				flusher.Flush()
			case msg := <-client.send:
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.event, msg.data)
				flusher.Flush()
			}
		}
	}
}

// ClientCount returns the number of active SSE clients.
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
