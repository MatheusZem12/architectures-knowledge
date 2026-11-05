// Package catalog owns products and stock. It reacts to events and never calls another module.
package catalog

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"eventdriven/events"
	"eventdriven/platform/broker"
	"eventdriven/platform/web"
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type Module struct {
	bus   *broker.Broker
	mu    sync.Mutex
	items map[string]Product
	seq   int
}

func New(bus *broker.Broker) *Module {
	m := &Module{bus: bus, items: map[string]Product{}}
	bus.Subscribe("catalog", m.handle)
	return m
}

func (m *Module) handle(event any) error {
	switch e := event.(type) {
	case events.CustomerConfirmed:
		m.reserve(e)
	case events.ReservationRejected:
		// R08: a refused coupon gives the stock back, and only then the rejection goes out
		m.release(e.Lines)
		m.bus.Publish(events.OrderRejected{RequestID: e.RequestID, Code: e.Code, Message: e.Message})
	case events.ReleaseRequested:
		m.release(e.Lines) // R13
		m.bus.Publish(events.StockReleased{RequestID: e.RequestID, OrderID: e.OrderID})
	}
	return nil
}

// reserve is all or nothing (R06) and records the current price (R07).
func (m *Module) reserve(e events.CustomerConfirmed) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, line := range e.Lines {
		product, ok := m.items[line.ProductID]
		if !ok {
			m.bus.Publish(events.OrderRejected{RequestID: e.RequestID, Code: "product_not_found", Message: "product does not exist"}) // R05
			return
		}
		if product.Stock < line.Quantity {
			m.bus.Publish(events.OrderRejected{RequestID: e.RequestID, Code: "insufficient_stock", Message: "insufficient stock"})
			return
		}
	}
	items := make([]events.Item, len(e.Lines))
	for i, line := range e.Lines {
		product := m.items[line.ProductID]
		product.Stock -= line.Quantity
		m.items[product.ID] = product
		items[i] = events.Item{ProductID: product.ID, Quantity: line.Quantity, UnitPriceCents: product.PriceCents}
	}
	m.bus.Publish(events.StockReserved{RequestID: e.RequestID, CustomerID: e.CustomerID, Items: items, Coupon: e.Coupon})
}

func (m *Module) release(lines []events.Line) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, line := range lines {
		if product, ok := m.items[line.ProductID]; ok {
			product.Stock += line.Quantity
			m.items[product.ID] = product
		}
	}
}

func (m *Module) create(p Product) (Product, error) {
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		return Product{}, web.Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	p.ID = fmt.Sprintf("prod-%d", m.seq)
	m.items[p.ID] = p
	return p, nil
}

func (m *Module) get(id string) (Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	product, ok := m.items[id]
	if !ok {
		return Product{}, web.Fail("not_found", "product not found")
	}
	return product, nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in Product
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.create(in))
		}
	})
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.get(r.PathValue("id")))
	})
}
