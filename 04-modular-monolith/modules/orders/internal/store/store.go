package store

import (
	"fmt"
	"sync"
)

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

type Store struct {
	mu    sync.RWMutex
	items map[string]Order
	seq   int
}

func New() *Store { return &Store{items: map[string]Order{}} }

func (s *Store) Insert(o Order) Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	o.ID = fmt.Sprintf("ord-%d", s.seq)
	s.items[o.ID] = o
	return o
}

func (s *Store) Find(id string) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.items[id]
	return o, ok
}

func (s *Store) Update(o Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[o.ID] = o
}

func (s *Store) HasActiveOrder(customerID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.items {
		if o.CustomerID == customerID && o.Status != "cancelled" {
			return true
		}
	}
	return false
}
