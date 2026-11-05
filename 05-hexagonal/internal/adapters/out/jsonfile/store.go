// Package jsonfile is a second outbound adapter: the data survives a restart.
//
// It is the proof of the architecture. It implements the same ports as memory, and to use it
// nothing in the core changed: just pick another adapter in main.go.
//
// It writes the whole file on every change. That is slow
// and not fit for production, but it is enough to show the swap.
package jsonfile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"sync"

	"hexagonal/internal/core/domain"
)

type snapshot struct {
	Seq           int
	Products      map[string]domain.Product
	Customers     map[string]domain.Customer
	Orders        map[string]domain.Order
	Notifications map[string][]domain.Notification
}

type Store struct {
	mu   sync.RWMutex
	path string
	data snapshot
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: snapshot{
		Products:      map[string]domain.Product{},
		Customers:     map[string]domain.Customer{},
		Orders:        map[string]domain.Order{},
		Notifications: map[string][]domain.Notification{},
	}}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}
	return s, nil
}

// persist writes to a temporary file and renames it, to never leave a half-written file.
func (s *Store) persist() {
	raw, err := json.Marshal(s.data)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(s.path+".tmp", raw, 0o644); err != nil {
		panic(err)
	}
	if err := os.Rename(s.path+".tmp", s.path); err != nil {
		panic(err)
	}
}

func (s *Store) nextID(prefix string) string {
	s.data.Seq++
	return prefix + "-" + strconv.Itoa(s.data.Seq)
}

func (s *Store) AddProduct(p domain.Product) domain.Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID("prod")
	s.data.Products[p.ID] = p
	s.persist()
	return p
}

func (s *Store) FindProduct(id string) (domain.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data.Products[id]
	return p, ok
}

func (s *Store) SaveProduct(p domain.Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Products[p.ID] = p
	s.persist()
}

func (s *Store) AddCustomer(c domain.Customer) domain.Customer {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.ID = s.nextID("cust")
	s.data.Customers[c.ID] = c
	s.persist()
	return c
}

func (s *Store) FindCustomer(id string) (domain.Customer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data.Customers[id]
	return c, ok
}

func (s *Store) EmailExists(email string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Customers {
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
	s.data.Orders[o.ID] = o
	s.persist()
	return o
}

func (s *Store) FindOrder(id string) (domain.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.data.Orders[id]
	return o, ok
}

func (s *Store) SaveOrder(o domain.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Orders[o.ID] = o
	s.persist()
}

func (s *Store) HasActiveOrder(customerID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.data.Orders {
		if o.CustomerID == customerID && o.Status != domain.StatusCancelled {
			return true
		}
	}
	return false
}

func (s *Store) AddNotification(n domain.Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Notifications[n.CustomerID] = append(s.data.Notifications[n.CustomerID], n)
	s.persist()
}

func (s *Store) NotificationsOf(customerID string) []domain.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Notification{}, s.data.Notifications[customerID]...)
}
