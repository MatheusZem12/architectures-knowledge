// Package domain rebuilds state from events. Current state is not stored anywhere:
// it is always the result of replaying what happened.
package domain

import "eventsourcing/events"

type Product struct {
	Exists     bool
	PriceCents int
	Stock      int
}

// ReplayProduct folds a product's events into its current state.
func ReplayProduct(history []any) Product {
	var product Product
	for _, event := range history {
		switch e := event.(type) {
		case events.ProductCreated:
			product = Product{Exists: true, PriceCents: e.PriceCents, Stock: e.Stock}
		case events.StockReserved:
			product.Stock -= e.Quantity
		case events.StockReleased:
			product.Stock += e.Quantity
		}
	}
	return product
}

type Customer struct {
	Exists       bool
	ActiveOrders int
}

// ReplayCustomer answers R05 and R08 from the customer's own history.
func ReplayCustomer(history []any) Customer {
	var customer Customer
	for _, event := range history {
		switch event.(type) {
		case events.CustomerRegistered:
			customer.Exists = true
		case events.CustomerOrderOpened:
			customer.ActiveOrders++
		case events.CustomerOrderClosed:
			customer.ActiveOrders--
		}
	}
	return customer
}

type Order struct {
	Exists     bool
	CustomerID string
	Status     string
	Items      []events.OrderItem
	TotalCents int
}

// ReplayOrder folds an order's events. The state machine in R11 is visible right here.
func ReplayOrder(history []any) Order {
	var order Order
	for _, event := range history {
		switch e := event.(type) {
		case events.OrderPlaced:
			order = Order{Exists: true, CustomerID: e.CustomerID, Status: "pending", Items: e.Items, TotalCents: e.TotalCents}
		case events.OrderPaid:
			order.Status = "paid"
		case events.OrderCancelled:
			order.Status = "cancelled"
		}
	}
	return order
}
