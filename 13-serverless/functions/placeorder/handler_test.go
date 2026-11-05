package placeorder

import (
	"testing"

	"serverless/cloud"
	"serverless/functions/shared"
)

// A function is just a call: no server to start, and only the managed services to set up.
func TestRefusedCouponLeavesTheStockAlone(t *testing.T) {
	env := cloud.NewEnv()
	_ = env.DB.Transact(func(tx *cloud.Tx) error {
		tx.Put("product:p", shared.Product{ID: "p", PriceCents: 100, Stock: 5})
		tx.Put("customer:c", shared.Customer{ID: "c"})
		return nil
	})

	response := Handler(env, cloud.Request{Body: []byte(`{"customer_id":"c","coupon":"NOPE","items":[{"product_id":"p","quantity":2}]}`)})

	if response.Status != 422 {
		t.Fatalf("status = %d", response.Status)
	}
	var product shared.Product
	_ = env.DB.Transact(func(tx *cloud.Tx) error {
		tx.Get("product:p", &product)
		return nil
	})
	if product.Stock != 5 {
		t.Fatalf("stock = %d, want 5", product.Stock)
	}
}
