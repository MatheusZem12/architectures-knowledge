package service

import (
	"testing"

	"layered/internal/integration"
	"layered/internal/model"
	"layered/internal/repository"
)

// To test the discount, you must assemble the real repositories and gateway: the service
// depends on concrete types, with no interface to swap. It is cheap here because everything is
// in memory. With a real database, this test would start to require the database.
func TestWelcomeCouponGivesTenPercent(t *testing.T) {
	products := repository.NewProductRepository()
	customers := repository.NewCustomerRepository()
	service := NewOrderService(
		repository.NewOrderRepository(), products, customers,
		repository.NewNotificationRepository(), integration.NewPaymentGateway(),
	)
	product := products.Create(model.Product{Name: "Keyboard", PriceCents: 30_000, Stock: 5})
	customer := customers.Create(model.Customer{Name: "Ana", Email: "ana@example.com"})

	order, err := service.Place(PlaceOrderInput{
		CustomerID: customer.ID,
		Coupon:     "WELCOME10",
		Items:      []ItemInput{{ProductID: product.ID, Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.DiscountCents != 3_000 || order.TotalCents != 27_000 {
		t.Fatalf("discount = %d, total = %d", order.DiscountCents, order.TotalCents)
	}
}
