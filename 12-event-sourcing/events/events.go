// Package events lists everything that can happen in the store. Events are facts in the past
// tense, and they are never changed or deleted: this is the system's only source of truth.
package events

import (
	"encoding/json"
	"fmt"
)

type ProductCreated struct {
	ProductID  string
	Name       string
	PriceCents int
	Stock      int
}

type StockReserved struct {
	ProductID string
	OrderID   string
	Quantity  int
}

type StockReleased struct {
	ProductID string
	OrderID   string
	Quantity  int
}

type CustomerRegistered struct {
	CustomerID string
	Name       string
	Email      string
}

type CustomerOrderOpened struct {
	CustomerID string
	OrderID    string
}

type CustomerOrderClosed struct {
	CustomerID string
	OrderID    string
}

type OrderItem struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

type OrderPlaced struct {
	OrderID       string
	CustomerID    string
	Items         []OrderItem
	Coupon        string
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

type OrderPaid struct {
	OrderID    string
	CustomerID string
}

type OrderCancelled struct {
	OrderID    string
	CustomerID string
}

// Name gives an event the type name stored in the log.
func Name(event any) string {
	switch event.(type) {
	case ProductCreated:
		return "ProductCreated"
	case StockReserved:
		return "StockReserved"
	case StockReleased:
		return "StockReleased"
	case CustomerRegistered:
		return "CustomerRegistered"
	case CustomerOrderOpened:
		return "CustomerOrderOpened"
	case CustomerOrderClosed:
		return "CustomerOrderClosed"
	case OrderPlaced:
		return "OrderPlaced"
	case OrderPaid:
		return "OrderPaid"
	case OrderCancelled:
		return "OrderCancelled"
	}
	panic(fmt.Sprintf("unknown event: %T", event))
}

// Decode turns a stored event back into its type. Old events live forever, so every type ever
// written must stay readable: this is the versioning problem of event sourcing.
func Decode(name string, data json.RawMessage) (any, error) {
	switch name {
	case "ProductCreated":
		return decode[ProductCreated](data)
	case "StockReserved":
		return decode[StockReserved](data)
	case "StockReleased":
		return decode[StockReleased](data)
	case "CustomerRegistered":
		return decode[CustomerRegistered](data)
	case "CustomerOrderOpened":
		return decode[CustomerOrderOpened](data)
	case "CustomerOrderClosed":
		return decode[CustomerOrderClosed](data)
	case "OrderPlaced":
		return decode[OrderPlaced](data)
	case "OrderPaid":
		return decode[OrderPaid](data)
	case "OrderCancelled":
		return decode[OrderCancelled](data)
	}
	return nil, fmt.Errorf("unknown event type in the log: %s", name)
}

func decode[T any](data json.RawMessage) (any, error) {
	var event T
	err := json.Unmarshal(data, &event)
	return event, err
}
