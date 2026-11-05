package orders

import (
	"testing"

	"byfeature/customers"
	"byfeature/notifications"
	"byfeature/products"
)

// Testing orders requires the concrete packages of the other features.
func TestBigOrderCouponGivesTwentyPercent(t *testing.T) {
	productStore, customerStore := products.NewStore(), customers.NewStore()
	service := NewService(productStore, customerStore, notifications.NewStore())
	product, _ := productStore.Create(products.Product{Name: "Monitor", PriceCents: 50_000, Stock: 1})
	customer, _ := customerStore.Create(customers.Customer{Name: "Ana", Email: "ana@example.com"})

	var in PlaceInput
	in.CustomerID, in.Coupon = customer.ID, "BIG20"
	in.Items = append(in.Items, struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}{product.ID, 1})

	order, err := service.Place(in)
	if err != nil {
		t.Fatal(err)
	}
	if order.DiscountCents != 10_000 {
		t.Fatalf("discount = %d", order.DiscountCents)
	}
}
