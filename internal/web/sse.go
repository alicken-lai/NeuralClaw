package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"neuralclaw/internal/observability"

	"go.uber.org/zap"
)

// SSEEvent represents a single event to flush to the client.
type SSEEvent struct {
	Type string
	Data interface{}
}

// SSEBroker manages client subscriptions mapping a topic (e.g. Run ID) to multiple clients.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan SSEEvent]struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string]map[chan SSEEvent]struct{}),
	}
}

func (b *SSEBroker) Subscribe(topic string) chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[chan SSEEvent]struct{})
	}

	ch := make(chan SSEEvent, 100) // Buffer to prevent blocking the broadcaster
	b.subscribers[topic][ch] = struct{}{}
	return ch
}

func (b *SSEBroker) Unsubscribe(topic string, ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if m, ok := b.subscribers[topic]; ok {
		delete(m, ch)
		close(ch)
		if len(m) == 0 {
			delete(b.subscribers, topic)
		}
	}
}

// Publish sends to all active channels tracking the given topic.
func (b *SSEBroker) Publish(topic string, eventType string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	m, ok := b.subscribers[topic]
	if !ok {
		return
	}

	evt := SSEEvent{Type: eventType, Data: data}

	for ch := range m {
		select {
		case ch <- evt:
		default:
			// Client's channel is full; we could drop the client here if wanted
			observability.Logger.Warn("SSE client buffer full, dropping event", zap.String("topic", topic))
		}
	}
}

// ServeHTTP implements the standard server-sent events connection handler.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request, topic string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := b.Subscribe(topic)
	defer b.Unsubscribe(topic, ch)

	// Send an initial connected ping
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return // Client disconnected
		case evt := <-ch:
			// Marshal payload
			payload, err := json.Marshal(evt.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, payload)
			flusher.Flush()
		}
	}
}
