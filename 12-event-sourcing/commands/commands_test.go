package commands

import (
	"path/filepath"
	"testing"

	"eventsourcing/domain"
	"eventsourcing/eventstore"
)

// Tests here read as given, when, then: given these events, this command adds those events.
func TestCancelAddsEventsAndDeletesNothing(t *testing.T) {
	store, _ := eventstore.Open("")
	handler := New(store)
	product, _ := handler.CreateProduct("Keyboard", 100, 5)
	customer, _ := handler.RegisterCustomer("Ana", "ana@example.com")
	order, err := handler.PlaceOrder(customer, []Line{{product, 2}}, "")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Len()

	if err := handler.CancelOrder(order); err != nil {
		t.Fatal(err)
	}

	if store.Len() <= before {
		t.Fatalf("the log must only grow: %d events before, %d after", before, store.Len())
	}
	if got := domain.ReplayProduct(store.Load(product)).Stock; got != 5 {
		t.Fatalf("stock = %d, want 5", got)
	}
}

// The central promise: delete everything except the log, and all the state comes back.
func TestRestartingRebuildsEverythingFromTheLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	first, err := eventstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(first)
	product, _ := handler.CreateProduct("Keyboard", 100, 5)
	customer, _ := handler.RegisterCustomer("Ana", "ana@example.com")
	if _, err := handler.PlaceOrder(customer, []Line{{product, 2}}, ""); err != nil {
		t.Fatal(err)
	}

	second, err := eventstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := domain.ReplayProduct(second.Load(product)).Stock; got != 3 {
		t.Fatalf("stock after restart = %d, want 3", got)
	}
	if _, err := New(second).RegisterCustomer("Other", "ANA@example.com"); err == nil {
		t.Fatal("R03 must still hold after a restart")
	}
}
