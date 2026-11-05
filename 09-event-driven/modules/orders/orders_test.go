package orders

import (
	"errors"
	"sync/atomic"
	"testing"

	"eventdriven/events"
	"eventdriven/platform/broker"
	"eventdriven/platform/web"
)

// A test here speaks the system's language: given these events, it expects those events.
// The other modules do not exist: a small subscriber plays their part.
func TestRefusedCouponPublishesTheCompensation(t *testing.T) {
	bus := broker.New(false)
	module := New(bus)
	var compensated atomic.Int32
	bus.Subscribe("fake-catalog", func(event any) error {
		switch e := event.(type) {
		case events.OrderRequested:
			bus.Publish(events.StockReserved{RequestID: e.RequestID, CustomerID: e.CustomerID, Coupon: e.Coupon,
				Items: []events.Item{{ProductID: "p", Quantity: 1, UnitPriceCents: 100}}})
		case events.ReservationRejected:
			compensated.Add(1)
			bus.Publish(events.OrderRejected{RequestID: e.RequestID, Code: e.Code, Message: e.Message})
		}
		return nil
	})

	var in PlaceInput
	in.CustomerID, in.Coupon = "c", "NOSUCHCOUPON"
	in.Items = append(in.Items, struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}{"p", 1})

	_, err := module.Place(in)

	var refusal *web.Error
	if !errors.As(err, &refusal) || refusal.Code != "invalid_coupon" {
		t.Fatalf("error = %v", err)
	}
	if compensated.Load() != 1 {
		t.Fatalf("compensation published %d times, want 1", compensated.Load())
	}
}

func TestDiscount(t *testing.T) {
	if got, err := discount("WELCOME10", 999, true); err != nil || got != 99 {
		t.Fatalf("discount = %d, error = %v", got, err)
	}
	if _, err := discount("WELCOME10", 999, false); err == nil || err.Code != "coupon_not_eligible" {
		t.Fatalf("error = %v", err)
	}
}
