package broker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if condition() {
			return
		}
	}
	t.Fatal("condition not reached in time")
}

// This is what folder 07 lacked: the consumer is down for a moment, and the event still arrives.
func TestBriefOutageDoesNotLoseTheEvent(t *testing.T) {
	bus := New(false)
	var attempts, delivered atomic.Int32
	bus.Subscribe("flaky", func(any) error {
		if attempts.Add(1) < 3 {
			return errors.New("temporarily down")
		}
		delivered.Add(1)
		return nil
	})

	bus.Publish("order_paid")

	waitFor(t, func() bool { return delivered.Load() == 1 })
	if len(bus.DeadLetters()) != 0 {
		t.Fatalf("dead letters = %v", bus.DeadLetters())
	}
}

func TestLongOutageGoesToDeadLetters(t *testing.T) {
	bus := New(false)
	bus.Subscribe("down", func(any) error { return errors.New("down") })

	bus.Publish("order_paid")

	waitFor(t, func() bool { return len(bus.DeadLetters()) == 1 })
	if got := bus.DeadLetters()[0]; got.Subscriber != "down" || got.Event != "order_paid" {
		t.Fatalf("dead letter = %+v", got)
	}
}

func TestSlowSubscriberHoldsNobodyBack(t *testing.T) {
	bus := New(false)
	release := make(chan struct{})
	defer close(release)
	var fast atomic.Int32
	bus.Subscribe("slow", func(any) error { <-release; return nil })
	bus.Subscribe("fast", func(any) error { fast.Add(1); return nil })

	bus.Publish("a")
	bus.Publish("b")

	waitFor(t, func() bool { return fast.Load() == 2 })
}
