// Package models is the M in MVC: the data and every business rule.
//
// It knows nothing about HTTP, JSON or HTML. This is the "fat model" style: with no service
// layer, the rules live on the model itself.
package models

import (
	"fmt"
	"sync"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func Fail(code, message string) *Error { return &Error{Code: code, Message: message} }

// PaymentGateway answers whether a charge was approved.
type PaymentGateway func(amountCents int) bool

const maxChargeCents = 1_000_000

// FakeGateway simulates a real gateway, which declines high amounts.
func FakeGateway(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}

// DB holds the data in memory. In a framework like Rails or Django, this role belongs to the ORM.
type DB struct {
	mu            sync.Mutex // in a real database, this would be a transaction
	products      map[string]Product
	customers     map[string]Customer
	orders        map[string]Order
	notifications map[string][]Notification
	gateway       PaymentGateway
	seq           int
}

func NewDB(gateway PaymentGateway) *DB {
	return &DB{
		products:      map[string]Product{},
		customers:     map[string]Customer{},
		orders:        map[string]Order{},
		notifications: map[string][]Notification{},
		gateway:       gateway,
	}
}

func (db *DB) nextID(prefix string) string {
	db.seq++
	return fmt.Sprintf("%s-%d", prefix, db.seq)
}
