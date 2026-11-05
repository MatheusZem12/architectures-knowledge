// Package projections builds read models from the log. They are disposable: delete one, replay
// the log, and it comes back. A new one can also be added later, and it will know the whole past.
package projections

import (
	"sync"

	"eventsourcing/events"
	"eventsourcing/eventstore"
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

type Item struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type Order struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	Items         []Item `json:"items"`
	Coupon        string `json:"coupon"`
	SubtotalCents int    `json:"subtotal_cents"`
	DiscountCents int    `json:"discount_cents"`
	ShippingCents int    `json:"shipping_cents"`
	TotalCents    int    `json:"total_cents"`
}

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Views struct {
	mu            sync.RWMutex
	products      map[string]Product
	customers     map[string]Customer
	orders        map[string]Order
	notifications map[string][]Notification
}

func New(store *eventstore.Store) *Views {
	v := &Views{
		products:      map[string]Product{},
		customers:     map[string]Customer{},
		orders:        map[string]Order{},
		notifications: map[string][]Notification{},
	}
	store.Subscribe(v.apply)
	return v
}

func (v *Views) apply(record eventstore.Record) {
	v.mu.Lock()
	defer v.mu.Unlock()
	switch e := record.Event.(type) {
	case events.ProductCreated:
		v.products[e.ProductID] = Product{e.ProductID, e.Name, e.PriceCents, e.Stock}
	case events.StockReserved:
		v.addStock(e.ProductID, -e.Quantity)
	case events.StockReleased:
		v.addStock(e.ProductID, e.Quantity)
	case events.CustomerRegistered:
		v.customers[e.CustomerID] = Customer{e.CustomerID, e.Name, e.Email}
	case events.OrderPlaced:
		items := make([]Item, len(e.Items))
		for i, item := range e.Items {
			items[i] = Item(item)
		}
		v.orders[e.OrderID] = Order{e.OrderID, e.CustomerID, "pending", items, e.Coupon, e.SubtotalCents, e.DiscountCents, e.ShippingCents, e.TotalCents}
		v.notify(e.CustomerID, e.OrderID, "order_created")
	case events.OrderPaid:
		v.setStatus(e.OrderID, "paid")
		v.notify(e.CustomerID, e.OrderID, "order_paid")
	case events.OrderCancelled:
		v.setStatus(e.OrderID, "cancelled")
		v.notify(e.CustomerID, e.OrderID, "order_cancelled")
	}
}

func (v *Views) addStock(productID string, quantity int) {
	product := v.products[productID]
	product.Stock += quantity
	v.products[productID] = product
}

func (v *Views) setStatus(orderID, status string) {
	order := v.orders[orderID]
	order.Status = status
	v.orders[orderID] = order
}

// R14: notifications are one more view of the same facts
func (v *Views) notify(customerID, orderID, kind string) {
	v.notifications[customerID] = append(v.notifications[customerID], Notification{customerID, orderID, kind})
}

func (v *Views) Product(id string) (Product, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	view, ok := v.products[id]
	return view, ok
}

func (v *Views) Customer(id string) (Customer, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	view, ok := v.customers[id]
	return view, ok
}

func (v *Views) Order(id string) (Order, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	view, ok := v.orders[id]
	return view, ok
}

func (v *Views) Notifications(customerID string) []Notification {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return append([]Notification{}, v.notifications[customerID]...)
}
