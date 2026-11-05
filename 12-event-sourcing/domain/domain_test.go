package domain

import (
	"testing"

	"eventsourcing/events"
)

// Time travel: replaying only part of the history gives the state at that moment.
func TestReplayingPartOfTheHistoryGivesThePast(t *testing.T) {
	history := []any{
		events.ProductCreated{ProductID: "p", PriceCents: 100, Stock: 10},
		events.StockReserved{ProductID: "p", OrderID: "o1", Quantity: 3},
		events.StockReserved{ProductID: "p", OrderID: "o2", Quantity: 2},
		events.StockReleased{ProductID: "p", OrderID: "o1", Quantity: 3},
	}
	want := []int{10, 7, 5, 8}
	for i, stock := range want {
		if got := ReplayProduct(history[:i+1]).Stock; got != stock {
			t.Errorf("after %d events: stock = %d, want %d", i+1, got, stock)
		}
	}
}
