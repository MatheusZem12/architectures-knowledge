// Package presenters converts use case output into the format the client sees.
// If an API v2 with different JSON appears tomorrow, only this changes.
package presenters

import (
	"errors"

	"cleanarch/entities"
	"cleanarch/usecases"
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type OrderItem struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type Order struct {
	ID            string      `json:"id"`
	CustomerID    string      `json:"customer_id"`
	Status        string      `json:"status"`
	Items         []OrderItem `json:"items"`
	Coupon        string      `json:"coupon"`
	SubtotalCents int         `json:"subtotal_cents"`
	DiscountCents int         `json:"discount_cents"`
	ShippingCents int         `json:"shipping_cents"`
	TotalCents    int         `json:"total_cents"`
}

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type ErrorView struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func PresentOrder(o usecases.OrderOutput) Order {
	items := make([]OrderItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItem(item)
	}
	return Order{o.ID, o.CustomerID, o.Status, items, o.Coupon, o.SubtotalCents, o.DiscountCents, o.ShippingCents, o.TotalCents}
}

func PresentNotifications(list []usecases.NotificationOutput) []Notification {
	out := make([]Notification, len(list))
	for i, n := range list {
		out[i] = Notification(n)
	}
	return out
}

// PresentError decides how each business error appears to the caller.
func PresentError(err error) (int, ErrorView) {
	var businessErr *entities.Error
	if !errors.As(err, &businessErr) {
		return 500, ErrorView{"internal", "internal error"}
	}
	return statusFor(businessErr.Code), ErrorView{businessErr.Code, businessErr.Message}
}

func statusFor(code string) int {
	switch code {
	case entities.CodeNotFound:
		return 404
	case entities.CodePaymentDeclined:
		return 402
	case entities.CodeEmailTaken, entities.CodeInsufficientStock, entities.CodeInvalidTransition:
		return 409
	default:
		return 422
	}
}
