package app

import (
	"errors"
	"testing"

	"hexagonal/internal/adapters/out/memory"
	"hexagonal/internal/core/domain"
)

// declineAll is a fake gateway, written in three lines, that declines everything. Since the core
// only knows the interface, the decline path can be tested without any "magic" value.
type declineAll struct{}

func (declineAll) Charge(int) bool { return false }

func TestDeclinedPaymentKeepsOrderPendingAndDoesNotNotify(t *testing.T) {
	store := memory.New()
	shop := NewShop(store, store, store, store, declineAll{})
	product, _ := shop.CreateProduct("Keyboard", 100, 1)
	customer, _ := shop.CreateCustomer("Ana", "ana@example.com")
	order, err := shop.PlaceOrder(customer.ID, []domain.Line{{ProductID: product.ID, Quantity: 1}}, "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = shop.PayOrder(order.ID)

	var businessErr *domain.Error
	if !errors.As(err, &businessErr) || businessErr.Code != domain.CodePaymentDeclined {
		t.Fatalf("error = %v", err)
	}
	saved, _ := shop.GetOrder(order.ID)
	if saved.Status != domain.StatusPending {
		t.Fatalf("status = %s", saved.Status)
	}
	if got := len(shop.Notifications(customer.ID)); got != 1 {
		t.Fatalf("notifications = %d, want only order_created", got)
	}
}
