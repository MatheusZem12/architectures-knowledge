package read

import (
	"testing"

	"cqrs/events"
)

// The read side is tested by feeding it facts. It needs no commands and no business rules.
func TestSummaryFollowsTheFacts(t *testing.T) {
	store := NewStore()
	store.Publish(events.CustomerRegistered{ID: "c", Name: "Ana", Email: "ana@example.com"})
	store.Publish(events.OrderPlaced{ID: "o", CustomerID: "c", TotalCents: 500})
	store.Publish(events.OrderPaid{ID: "o"})

	if got, _ := store.Summary("c"); got.ActiveOrders != 1 || got.PaidCents != 500 {
		t.Fatalf("after paying: %+v", got)
	}

	store.Publish(events.OrderCancelled{ID: "o"})

	if got, _ := store.Summary("c"); got.ActiveOrders != 0 || got.PaidCents != 0 {
		t.Fatalf("after cancelling: %+v", got)
	}
}
