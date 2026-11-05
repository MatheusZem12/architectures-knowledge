// Package events is the bridge between the sides: the write side says what happened, and the
// read side builds its views from that. Neither side imports the other.
package events

type ProductCreated struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

type StockChanged struct {
	ProductID string
	Stock     int
}

type CustomerRegistered struct {
	ID    string
	Name  string
	Email string
}

type OrderItem struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

type OrderPlaced struct {
	ID            string
	CustomerID    string
	Items         []OrderItem
	Coupon        string
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

type OrderPaid struct {
	ID string
}

type OrderCancelled struct {
	ID string
}
