// Package read is the query side: it answers questions, and never decides anything.
//
// Each view is already in the shape the client wants, so a query is just a lookup. Nothing here
// can refuse an event: when it arrives, the fact has already happened.
package read

import (
	"sync"

	"cqrs/events"
)

type ProductView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type CustomerView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type OrderItemView struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type OrderView struct {
	ID            string          `json:"id"`
	CustomerID    string          `json:"customer_id"`
	Status        string          `json:"status"`
	Items         []OrderItemView `json:"items"`
	Coupon        string          `json:"coupon"`
	SubtotalCents int             `json:"subtotal_cents"`
	DiscountCents int             `json:"discount_cents"`
	ShippingCents int             `json:"shipping_cents"`
	TotalCents    int             `json:"total_cents"`
}

type NotificationView struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

// CustomerSummary exists only on this side, and only for reading. In the other architectures,
// answering this would mean scanning every order. Here, it is kept up to date as events arrive.
type CustomerSummary struct {
	CustomerID   string `json:"customer_id"`
	ActiveOrders int    `json:"active_orders"`
	PaidCents    int    `json:"paid_cents"`
}

type Store struct {
	mu            sync.RWMutex
	products      map[string]ProductView
	customers     map[string]CustomerView
	orders        map[string]OrderView
	notifications map[string][]NotificationView
	summaries     map[string]CustomerSummary
}

func NewStore() *Store {
	return &Store{
		products:      map[string]ProductView{},
		customers:     map[string]CustomerView{},
		orders:        map[string]OrderView{},
		notifications: map[string][]NotificationView{},
		summaries:     map[string]CustomerSummary{},
	}
}

// Publish is the projection: it turns each fact into updates to the views.
func (s *Store) Publish(event any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch e := event.(type) {
	case events.ProductCreated:
		s.products[e.ID] = ProductView(e)
	case events.StockChanged:
		product := s.products[e.ProductID]
		product.Stock = e.Stock
		s.products[e.ProductID] = product
	case events.CustomerRegistered:
		s.customers[e.ID] = CustomerView(e)
		s.summaries[e.ID] = CustomerSummary{CustomerID: e.ID}
	case events.OrderPlaced:
		s.applyPlaced(e)
	case events.OrderPaid:
		s.applyPaid(e)
	case events.OrderCancelled:
		s.applyCancelled(e)
	}
}

func (s *Store) applyPlaced(e events.OrderPlaced) {
	items := make([]OrderItemView, len(e.Items))
	for i, item := range e.Items {
		items[i] = OrderItemView(item)
	}
	s.orders[e.ID] = OrderView{e.ID, e.CustomerID, "pending", items, e.Coupon, e.SubtotalCents, e.DiscountCents, e.ShippingCents, e.TotalCents}
	summary := s.summaries[e.CustomerID]
	summary.ActiveOrders++
	s.summaries[e.CustomerID] = summary
	s.notify(e.CustomerID, e.ID, "order_created")
}

func (s *Store) applyPaid(e events.OrderPaid) {
	order := s.orders[e.ID]
	order.Status = "paid"
	s.orders[e.ID] = order
	summary := s.summaries[order.CustomerID]
	summary.PaidCents += order.TotalCents
	s.summaries[order.CustomerID] = summary
	s.notify(order.CustomerID, e.ID, "order_paid")
}

func (s *Store) applyCancelled(e events.OrderCancelled) {
	order := s.orders[e.ID]
	summary := s.summaries[order.CustomerID]
	summary.ActiveOrders--
	if order.Status == "paid" {
		summary.PaidCents -= order.TotalCents
	}
	s.summaries[order.CustomerID] = summary
	order.Status = "cancelled"
	s.orders[e.ID] = order
	s.notify(order.CustomerID, e.ID, "order_cancelled")
}

// R14: notifications are just one more view built from the facts
func (s *Store) notify(customerID, orderID, kind string) {
	s.notifications[customerID] = append(s.notifications[customerID], NotificationView{customerID, orderID, kind})
}

// ---------- queries ----------

func (s *Store) Product(id string) (ProductView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	view, ok := s.products[id]
	return view, ok
}

func (s *Store) Customer(id string) (CustomerView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	view, ok := s.customers[id]
	return view, ok
}

func (s *Store) Order(id string) (OrderView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	view, ok := s.orders[id]
	return view, ok
}

func (s *Store) Notifications(customerID string) []NotificationView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]NotificationView{}, s.notifications[customerID]...)
}

func (s *Store) Summary(customerID string) (CustomerSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	view, ok := s.summaries[customerID]
	return view, ok
}
