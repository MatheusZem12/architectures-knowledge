// Package store is what the slices share: the data, and no rule at all.
//
// A pragmatic choice: these types carry JSON tags and are returned directly by several slices.
// A stricter version would give each slice its own response type, at the cost of more repetition.
package store

import (
	"fmt"
	"sync"
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

type OrderItem struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type Order struct {
	ID            string      `json:"id"`
	CustomerID    string      `json:"customer_id"`
	Status        string      `json:"status"`
	Items         []OrderItem `json:"items"`
	Coupon        string      `json:"coupon"`
	SubtotalCents int         `json:"subtotal_cents"`
	DiscountCents int         `json:"discount_cents"`
	ShippingCents int         `json:"shipping_cents"`
	TotalCents    int         `json:"total_cents"`
}

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

// DB plays the part of a database: each slice writes its own "query" inside Atomic.
type DB struct {
	mu            sync.Mutex
	Products      map[string]Product
	Customers     map[string]Customer
	Orders        map[string]Order
	Notifications map[string][]Notification
	seq           int
}

func New() *DB {
	return &DB{
		Products:      map[string]Product{},
		Customers:     map[string]Customer{},
		Orders:        map[string]Order{},
		Notifications: map[string][]Notification{},
	}
}

// Atomic runs fn as one transaction. Every access to the data goes through it.
func (db *DB) Atomic(fn func() error) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return fn()
}

func (db *DB) NextID(prefix string) string {
	db.seq++
	return fmt.Sprintf("%s-%d", prefix, db.seq)
}

func (db *DB) Notify(order Order, kind string) {
	db.Notifications[order.CustomerID] = append(db.Notifications[order.CustomerID], Notification{order.CustomerID, order.ID, kind})
}
