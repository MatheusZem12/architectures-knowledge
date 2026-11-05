package domain

import "testing"

// Domain test: no server, no repository, no fake. Just the rule.
func TestShippingIsCalculatedAfterDiscount(t *testing.T) {
	items := []OrderItem{{ProductID: "p", Quantity: 1, UnitPriceCents: 21_000}}
	order, err := NewOrder("c", items, "WELCOME10", true)
	if err != nil {
		t.Fatal(err)
	}
	if order.DiscountCents != 2_100 || order.ShippingCents != 2_000 || order.TotalCents != 20_900 {
		t.Fatalf("unexpected order: %+v", order)
	}
}

func TestCancelledOrderIsFinal(t *testing.T) {
	order := Order{Status: StatusCancelled}
	if order.CanPay() == nil || order.Cancel() == nil {
		t.Fatal("a cancelled order cannot change state")
	}
}
