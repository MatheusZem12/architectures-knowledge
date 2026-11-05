// Package notifications reacts to order events.
//
// The orders module does not even know this module exists: deleting this folder (and the
// matching lines in main.go) does not require changing a single line of orders.
package notifications

import (
	"net/http"

	"modular/modules/notifications/internal/store"
	"modular/modules/orders"
	"modular/platform/events"
	"modular/platform/web"
)

type Module struct {
	store *store.Store
}

func New(bus *events.Bus) *Module {
	m := &Module{store: store.New()}
	bus.Subscribe(m.onEvent)
	return m
}

// R14
func (m *Module) onEvent(event any) {
	if e, ok := event.(orders.Event); ok {
		m.store.Add(store.Notification{CustomerID: e.CustomerID, OrderID: e.OrderID, Type: e.Type})
	}
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, 200, m.store.ByCustomer(r.PathValue("id")))
	})
}
