package store

import "sync"

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string][]Notification
}

func New() *Store { return &Store{items: map[string][]Notification{}} }

func (s *Store) Add(n Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[n.CustomerID] = append(s.items[n.CustomerID], n)
}

func (s *Store) ByCustomer(customerID string) []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Notification{}, s.items[customerID]...)
}
