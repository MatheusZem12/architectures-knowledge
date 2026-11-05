// Package events is an in-memory bus: publishers do not know who is listening.
//
// It is synchronous on purpose, to stay simple and predictable in a single process. Swapping it for
// a real broker (Kafka, RabbitMQ) is precisely one of the steps to extract a module.
package events

import "sync"

type Handler func(event any)

type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

func NewBus() *Bus { return &Bus{} }

func (b *Bus) Subscribe(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

func (b *Bus) Publish(event any) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, handler := range b.handlers {
		handler(event)
	}
}
