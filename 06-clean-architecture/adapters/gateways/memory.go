// Package gateways implements the interfaces the use cases declare.
package gateways

import (
	"fmt"
	"strings"
	"sync"

	"cleanarch/entities"
)

type MemoryDB struct {
	tx            sync.Mutex
	mu            sync.RWMutex
	products      map[string]entities.Product
	customers     map[string]entities.Customer
	orders        map[string]entities.Order
	notifications map[string][]entities.Notification
	seq           int
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		products:      map[string]entities.Product{},
		customers:     map[string]entities.Customer{},
		orders:        map[string]entities.Order{},
		notifications: map[string][]entities.Notification{},
	}
}

// Atomic plays the role of BEGIN and COMMIT: atomic operations run one at a time.
func (db *MemoryDB) Atomic(fn func() error) error {
	db.tx.Lock()
	defer db.tx.Unlock()
	return fn()
}

func (db *MemoryDB) nextID(prefix string) string {
	db.seq++
	return fmt.Sprintf("%s-%d", prefix, db.seq)
}

func (db *MemoryDB) AddProduct(p entities.Product) entities.Product {
	db.mu.Lock()
	defer db.mu.Unlock()
	p.ID = db.nextID("prod")
	db.products[p.ID] = p
	return p
}

func (db *MemoryDB) FindProduct(id string) (entities.Product, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	p, ok := db.products[id]
	return p, ok
}

func (db *MemoryDB) SaveProduct(p entities.Product) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.products[p.ID] = p
}

func (db *MemoryDB) AddCustomer(c entities.Customer) entities.Customer {
	db.mu.Lock()
	defer db.mu.Unlock()
	c.ID = db.nextID("cust")
	db.customers[c.ID] = c
	return c
}

func (db *MemoryDB) FindCustomer(id string) (entities.Customer, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	c, ok := db.customers[id]
	return c, ok
}

func (db *MemoryDB) EmailExists(email string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, c := range db.customers {
		if strings.EqualFold(c.Email, email) {
			return true
		}
	}
	return false
}

func (db *MemoryDB) AddOrder(o entities.Order) entities.Order {
	db.mu.Lock()
	defer db.mu.Unlock()
	o.ID = db.nextID("ord")
	db.orders[o.ID] = o
	return o
}

func (db *MemoryDB) FindOrder(id string) (entities.Order, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	o, ok := db.orders[id]
	return o, ok
}

func (db *MemoryDB) SaveOrder(o entities.Order) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.orders[o.ID] = o
}

func (db *MemoryDB) HasActiveOrder(customerID string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, o := range db.orders {
		if o.CustomerID == customerID && o.Status != entities.StatusCancelled {
			return true
		}
	}
	return false
}

func (db *MemoryDB) AddNotification(n entities.Notification) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.notifications[n.CustomerID] = append(db.notifications[n.CustomerID], n)
}

func (db *MemoryDB) NotificationsOf(customerID string) []entities.Notification {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return append([]entities.Notification{}, db.notifications[customerID]...)
}
