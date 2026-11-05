// Package notifications stores and lists the notices sent to customers.
package notifications

import (
	"net/http"
	"sync"

	"byfeature/web"
)

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string][]Notification
}

func NewStore() *Store { return &Store{items: map[string][]Notification{}} }

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

func Register(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, 200, store.ByCustomer(r.PathValue("id")))
	})
}
