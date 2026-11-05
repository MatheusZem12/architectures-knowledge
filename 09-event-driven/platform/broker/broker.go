// Package broker is an in-memory stand-in for Kafka or RabbitMQ.
//
// It has the two properties that matter here. Each subscriber has its own queue, so a slow one
// holds nobody else back. And a failed delivery is retried, so a brief outage does not lose the
// event. Unlike a real broker, it does not persist anything: if the process dies, the queues die too.
package broker

import (
	"log"
	"sync"
	"time"
)

const (
	maxAttempts = 5
	retryDelay  = 20 * time.Millisecond
)

// Handler receives every published event, and ignores the ones it does not care about.
type Handler func(event any) error

type DeadLetter struct {
	Subscriber string
	Event      any
	Err        error
}

type Broker struct {
	trace bool

	mu          sync.Mutex
	subscribers []*subscriber
	deadLetters []DeadLetter
}

type subscriber struct {
	name    string
	handler Handler
	mu      sync.Mutex
	wake    *sync.Cond
	queue   []any
}

func New(trace bool) *Broker { return &Broker{trace: trace} }

// Subscribe starts a goroutine that delivers this subscriber's events, in order, one at a time.
func (b *Broker) Subscribe(name string, handler Handler) {
	s := &subscriber{name: name, handler: handler}
	s.wake = sync.NewCond(&s.mu)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, s)
	b.mu.Unlock()
	go b.run(s)
}

// Publish only adds the event to the queues, and never waits for anyone to process it.
func (b *Broker) Publish(event any) {
	if b.trace {
		log.Printf("event: %T %+v", event, event)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.subscribers {
		s.mu.Lock()
		s.queue = append(s.queue, event)
		s.mu.Unlock()
		s.wake.Signal()
	}
}

func (b *Broker) run(s *subscriber) {
	for {
		s.mu.Lock()
		for len(s.queue) == 0 {
			s.wake.Wait()
		}
		event := s.queue[0]
		s.queue = s.queue[1:]
		s.mu.Unlock()
		b.deliver(s, event)
	}
}

func (b *Broker) deliver(s *subscriber, event any) {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = s.handler(event); err == nil {
			return
		}
		time.Sleep(retryDelay)
	}
	log.Printf("dead letter: %s could not handle %T: %v", s.name, event, err)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.deadLetters = append(b.deadLetters, DeadLetter{s.name, event, err})
}

// DeadLetters lists what failed every attempt. It is kept, not lost: someone can inspect and replay it.
func (b *Broker) DeadLetters() []DeadLetter {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]DeadLetter{}, b.deadLetters...)
}
