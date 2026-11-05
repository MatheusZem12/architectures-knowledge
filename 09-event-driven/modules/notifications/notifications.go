// Package notifications reacts to order facts.
//
// No module knows this one exists. Deleting this folder and its two lines in main.go does not
// change anything else, and a new consumer (say, an audit log) would plug in the same way.
package notifications

import (
	"net/http"
	"sync"

	"eventdriven/events"
	"eventdriven/platform/broker"
	"eventdriven/platform/web"
)

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Module struct {
	mu    sync.RWMutex
	items map[string][]Notification
}

func New(bus *broker.Broker) *Module {
	m := &Module{items: map[string][]Notification{}}
	bus.Subscribe("notifications", m.handle)
	return m
}

// R14
func (m *Module) handle(event any) error {
	switch e := event.(type) {
	case events.OrderPlaced:
		m.add(e.CustomerID, e.OrderID, "order_created")
	case events.OrderPaid:
		m.add(e.CustomerID, e.OrderID, "order_paid")
	case events.OrderCancelled:
		m.add(e.CustomerID, e.OrderID, "order_cancelled")
	}
	return nil
}

func (m *Module) add(customerID, orderID, kind string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[customerID] = append(m.items[customerID], Notification{customerID, orderID, kind})
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		web.JSON(w, 200, append([]Notification{}, m.items[r.PathValue("id")]...))
	})
}
