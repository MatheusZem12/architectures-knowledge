package placeorder

import (
	"errors"
	"testing"

	"verticalslice/store"
	"verticalslice/web"
)

// The slice is tested alone. The only thing to set up is the data it reads.
func TestWelcomeCouponOnlyOnFirstOrder(t *testing.T) {
	db := store.New()
	db.Products["p"] = store.Product{ID: "p", PriceCents: 30_000, Stock: 5}
	db.Customers["c"] = store.Customer{ID: "c"}
	var in request
	in.CustomerID, in.Coupon = "c", "WELCOME10"
	in.Items = append(in.Items, struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}{"p", 1})

	first, err := handle(db, in)
	if err != nil || first.DiscountCents != 3_000 {
		t.Fatalf("order = %+v, error = %v", first, err)
	}

	_, err = handle(db, in)
	var refusal *web.Error
	if !errors.As(err, &refusal) || refusal.Code != "coupon_not_eligible" {
		t.Fatalf("error = %v", err)
	}
}
