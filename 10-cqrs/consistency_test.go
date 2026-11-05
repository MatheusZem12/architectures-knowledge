package main

import (
	"testing"

	"cqrs/read"
	"cqrs/write"
)

// delayed holds the events back, like a queue between the sides would on a real system.
type delayed struct {
	target  *read.Store
	pending []any
}

func (d *delayed) Publish(event any) { d.pending = append(d.pending, event) }

func (d *delayed) flush() {
	for _, event := range d.pending {
		d.target.Publish(event)
	}
	d.pending = nil
}

// This folder delivers events synchronously, so reads are always up to date. This test shows
// what the architecture costs once a queue sits between the sides: the write succeeded, and the
// read does not know yet. It is why the contract suite would fail on an asynchronous projection.
func TestWithAQueueBetweenTheSidesReadsAreStale(t *testing.T) {
	views := read.NewStore()
	queue := &delayed{target: views}
	commands := write.NewHandler(queue, write.FakeGateway{})

	id, err := commands.CreateProduct(write.CreateProduct{Name: "Keyboard", PriceCents: 100, Stock: 1})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := views.Product(id); ok {
		t.Fatal("the read side should not know about the product yet")
	}

	queue.flush()

	if _, ok := views.Product(id); !ok {
		t.Fatal("the read side should know about the product after the events arrive")
	}
}
