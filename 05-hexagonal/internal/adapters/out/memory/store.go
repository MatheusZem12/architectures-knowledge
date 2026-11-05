// Package memory is an outbound adapter: it implements the repositories with in-memory maps.
package memory

import (
	"fmt"
	"strings"
	"sync"

	"hexagonal/internal/core/domain"
)

type Store struct {
	mu            sync.RWMutex
	products      map[string]domain.Product
	customers     map[string]domain.Customer
	orders        map[string]domain.Order
	notifications map[string][]domain.Notification
	seq           int
}

func New() *Store {
	return &Store{
		products:      map[string]domain.Product{},
		customers:     map[string]domain.Customer{},
		orders:        map[string]domain.Order{},
		notifications: map[string][]domain.Notification{},
	}
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%d", prefix, s.seq)
}

func (s *Store) AddProduct(p domain.Product) domain.Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID("prod")
	s.products[p.ID] = p
	return p
}

func (s *Store) FindProduct(id string) (domain.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *Store) SaveProduct(p domain.Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products[p.ID] = p
}

func (s *Store) AddCustomer(c domain.Customer) domain.Customer {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.ID = s.nextID("cust")
	s.customers[c.ID] = c
	return c
}

func (s *Store) FindCustomer(id string) (domain.Customer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.customers[id]
	return c, ok
}

func (s *Store) EmailExists(email string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.customers {
		if strings.EqualFold(c.Email, email) {
			return true
		}
	}
	return false
}

func (s *Store) AddOrder(o domain.Order) domain.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	o.ID = s.nextID("ord")
	s.orders[o.ID] = o
	return o
}

func (s *Store) FindOrder(id string) (domain.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *Store) SaveOrder(o domain.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[o.ID] = o
}

func (s *Store) HasActiveOrder(customerID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.orders {
		if o.CustomerID == customerID && o.Status != domain.StatusCancelled {
			return true
		}
	}
	return false
}

func (s *Store) AddNotification(n domain.Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[n.CustomerID] = append(s.notifications[n.CustomerID], n)
}

func (s *Store) NotificationsOf(customerID string) []domain.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Notification{}, s.notifications[customerID]...)
}
