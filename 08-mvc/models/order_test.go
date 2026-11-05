package models

import (
	"errors"
	"testing"
)

// The model is tested without HTTP: it has never heard of a request.
func TestDeclinedPaymentKeepsOrderPending(t *testing.T) {
	db := NewDB(func(int) bool { return false })
	product, _ := db.CreateProduct("Keyboard", 100, 1)
	customer, _ := db.CreateCustomer("Ana", "ana@example.com")
	order, err := db.PlaceOrder(customer.ID, []Line{{product.ID, 1}}, "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.PayOrder(order.ID)

	var businessErr *Error
	if !errors.As(err, &businessErr) || businessErr.Code != "payment_declined" {
		t.Fatalf("error = %v", err)
	}
	if saved, _ := db.FindOrder(order.ID); saved.Status != StatusPending {
		t.Fatalf("status = %s", saved.Status)
	}
}

func TestDiscountIsPure(t *testing.T) {
	if got, err := Discount("BIG20", 50_000, false); err != nil || got != 10_000 {
		t.Fatalf("discount = %d, error = %v", got, err)
	}
}
