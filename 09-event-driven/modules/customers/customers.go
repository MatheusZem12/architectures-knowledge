// Package customers owns customer data, and confirms whether an order's customer exists.
package customers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"eventdriven/events"
	"eventdriven/platform/broker"
	"eventdriven/platform/web"
)

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Module struct {
	bus   *broker.Broker
	mu    sync.Mutex
	items map[string]Customer
	seq   int
}

func New(bus *broker.Broker) *Module {
	m := &Module{bus: bus, items: map[string]Customer{}}
	bus.Subscribe("customers", m.handle)
	return m
}

func (m *Module) handle(event any) error {
	e, ok := event.(events.OrderRequested)
	if !ok {
		return nil
	}
	// R05
	if _, err := m.get(e.CustomerID); err != nil {
		m.bus.Publish(events.OrderRejected{RequestID: e.RequestID, Code: "customer_not_found", Message: "customer does not exist"})
		return nil
	}
	m.bus.Publish(events.CustomerConfirmed{RequestID: e.RequestID, CustomerID: e.CustomerID, Lines: e.Lines, Coupon: e.Coupon})
	return nil
}

func (m *Module) create(c Customer) (Customer, error) {
	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return Customer{}, web.Fail("validation", "customer needs a name and a valid email")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// R03
	for _, existing := range m.items {
		if strings.EqualFold(existing.Email, c.Email) {
			return Customer{}, web.Fail("email_taken", "email already registered")
		}
	}
	m.seq++
	c.ID = fmt.Sprintf("cust-%d", m.seq)
	m.items[c.ID] = c
	return c, nil
}

func (m *Module) get(id string) (Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	customer, ok := m.items[id]
	if !ok {
		return Customer{}, web.Fail("not_found", "customer not found")
	}
	return customer, nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in Customer
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.create(in))
		}
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.get(r.PathValue("id")))
	})
}
