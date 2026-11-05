// Package model holds the structs used by all layers.
//
// Notice there is no behavior here: only data. This is the "anemic model", typical
// of this architecture. The rules live in the services.
package model

const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusCancelled = "cancelled"
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
