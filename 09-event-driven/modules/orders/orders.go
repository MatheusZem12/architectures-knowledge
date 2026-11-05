// Package orders owns orders. It reacts to events and publishes new ones.
package orders

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"eventdriven/events"
	"eventdriven/platform/broker"
	"eventdriven/platform/web"
)

const replyTimeout = 2 * time.Second

type Item struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type Order struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	Items         []Item `json:"items"`
	Coupon        string `json:"coupon"`
	SubtotalCents int    `json:"subtotal_cents"`
	DiscountCents int    `json:"discount_cents"`
	ShippingCents int    `json:"shipping_cents"`
	TotalCents    int    `json:"total_cents"`
}

type outcome struct {
	order Order
	err   error
}

type Module struct {
	bus      *broker.Broker
	requests atomic.Int64

	mu      sync.RWMutex
	items   map[string]Order
	seq     int
	busy    map[string]bool // orders in the middle of a pay or cancel chain
	waiters map[string]chan outcome
}

func New(bus *broker.Broker) *Module {
	m := &Module{bus: bus, items: map[string]Order{}, busy: map[string]bool{}, waiters: map[string]chan outcome{}}
	bus.Subscribe("orders", m.handle)
	return m
}

func (m *Module) handle(event any) error {
	switch e := event.(type) {
	case events.StockReserved:
		m.price(e)
	case events.OrderRejected:
		m.reply(e.RequestID, outcome{err: web.Fail(e.Code, e.Message)})
	case events.PayRequested:
		m.startPayment(e)
	case events.PaymentApproved:
		m.finishPayment(e)
	case events.PaymentDeclined:
		m.clearBusy(e.OrderID)
		m.reply(e.RequestID, outcome{err: web.Fail("payment_declined", "payment declined")})
	case events.CancelRequested:
		m.startCancel(e)
	case events.StockReleased:
		m.finishCancel(e)
	}
	return nil
}

func (m *Module) price(e events.StockReserved) {
	order := Order{CustomerID: e.CustomerID, Status: "pending", Coupon: e.Coupon}
	lines := make([]events.Line, len(e.Items))
	for i, item := range e.Items {
		order.Items = append(order.Items, Item(item))
		order.SubtotalCents += item.UnitPriceCents * item.Quantity
		lines[i] = events.Line{ProductID: item.ProductID, Quantity: item.Quantity}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	value, refusal := discount(e.Coupon, order.SubtotalCents, !m.hasActiveOrder(e.CustomerID))
	if refusal != nil {
		// R08: the stock was already reserved, so the compensation has to go out
		m.bus.Publish(events.ReservationRejected{RequestID: e.RequestID, Lines: lines, Code: refusal.Code, Message: refusal.Message})
		return
	}
	order.DiscountCents = value
	order.ShippingCents = shipping(order.SubtotalCents - order.DiscountCents)
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	m.seq++
	order.ID = fmt.Sprintf("ord-%d", m.seq)
	m.items[order.ID] = order
	m.bus.Publish(events.OrderPlaced{OrderID: order.ID, CustomerID: order.CustomerID})
	m.replyLocked(e.RequestID, outcome{order: order})
}

func (m *Module) hasActiveOrder(customerID string) bool {
	for _, order := range m.items {
		if order.CustomerID == customerID && order.Status != "cancelled" {
			return true
		}
	}
	return false
}

func (m *Module) startPayment(e events.PayRequested) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.items[e.OrderID]
	if !ok {
		m.replyLocked(e.RequestID, outcome{err: web.Fail("not_found", "order not found")})
		return
	}
	// R11: an order in the middle of another chain cannot start a new one
	if order.Status != "pending" || m.busy[e.OrderID] {
		m.replyLocked(e.RequestID, outcome{err: web.Fail("invalid_transition", "only a pending order can be paid")})
		return
	}
	m.busy[e.OrderID] = true
	m.bus.Publish(events.ChargeRequested{RequestID: e.RequestID, OrderID: e.OrderID, AmountCents: order.TotalCents})
}

func (m *Module) finishPayment(e events.PaymentApproved) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := m.items[e.OrderID]
	order.Status = "paid"
	m.items[e.OrderID] = order
	delete(m.busy, e.OrderID)
	m.bus.Publish(events.OrderPaid{OrderID: order.ID, CustomerID: order.CustomerID})
	m.replyLocked(e.RequestID, outcome{order: order})
}

func (m *Module) startCancel(e events.CancelRequested) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.items[e.OrderID]
	if !ok {
		m.replyLocked(e.RequestID, outcome{err: web.Fail("not_found", "order not found")})
		return
	}
	// R11
	if order.Status == "cancelled" || m.busy[e.OrderID] {
		m.replyLocked(e.RequestID, outcome{err: web.Fail("invalid_transition", "order already cancelled")})
		return
	}
	m.busy[e.OrderID] = true
	lines := make([]events.Line, len(order.Items))
	for i, item := range order.Items {
		lines[i] = events.Line{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	m.bus.Publish(events.ReleaseRequested{RequestID: e.RequestID, OrderID: e.OrderID, Lines: lines})
}

func (m *Module) finishCancel(e events.StockReleased) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := m.items[e.OrderID]
	order.Status = "cancelled"
	m.items[e.OrderID] = order
	delete(m.busy, e.OrderID)
	m.bus.Publish(events.OrderCancelled{OrderID: order.ID, CustomerID: order.CustomerID})
	m.replyLocked(e.RequestID, outcome{order: order})
}

func (m *Module) clearBusy(orderID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.busy, orderID)
}

func (m *Module) reply(requestID string, result outcome) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replyLocked(requestID, result)
}

func (m *Module) replyLocked(requestID string, result outcome) {
	if waiter, ok := m.waiters[requestID]; ok {
		waiter <- result // buffered: never blocks, even if the request already timed out
		delete(m.waiters, requestID)
	}
}

// ask bridges the two worlds: HTTP wants an answer now, and events only give one later.
// It publishes the first event of a chain and waits for the outcome linked by the RequestID.
func (m *Module) ask(publish func(requestID string) any) (Order, error) {
	requestID := fmt.Sprintf("req-%d", m.requests.Add(1))
	waiter := make(chan outcome, 1)
	m.mu.Lock()
	m.waiters[requestID] = waiter
	m.mu.Unlock()

	m.bus.Publish(publish(requestID))

	select {
	case result := <-waiter:
		return result.order, result.err
	case <-time.After(replyTimeout):
		m.mu.Lock()
		delete(m.waiters, requestID)
		m.mu.Unlock()
		return Order{}, web.Fail("service_unavailable", "no answer in time")
	}
}

// PlaceInput is what a client sends to create an order.
type PlaceInput struct {
	CustomerID string `json:"customer_id"`
	Coupon     string `json:"coupon"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func (m *Module) Place(in PlaceInput) (Order, error) {
	// R04: a simple check of the request itself does not need to become an event
	if len(in.Items) == 0 {
		return Order{}, web.Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	lines := make([]events.Line, 0, len(in.Items))
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return Order{}, web.Fail("validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
		lines = append(lines, events.Line{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	return m.ask(func(requestID string) any {
		return events.OrderRequested{RequestID: requestID, CustomerID: in.CustomerID, Lines: lines, Coupon: in.Coupon}
	})
}

func (m *Module) get(id string) (Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	order, ok := m.items[id]
	if !ok {
		return Order{}, web.Fail("not_found", "order not found")
	}
	return order, nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in PlaceInput
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.Place(in))
		}
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.get(r.PathValue("id")))
	})
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		web.Reply(w, 200)(m.ask(func(requestID string) any { return events.PayRequested{RequestID: requestID, OrderID: id} }))
	})
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		web.Reply(w, 200)(m.ask(func(requestID string) any { return events.CancelRequested{RequestID: requestID, OrderID: id} }))
	})
}
