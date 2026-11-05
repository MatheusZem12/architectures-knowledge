// Package orders is the orders module.
//
// Compare these imports with those of orders in folder 03: only the public APIs of
// catalog, customers and payments come in. No store, no internal struct. Stock only changes
// by asking catalog (Reserve/Release), never from the outside.
package orders

import (
	"errors"
	"net/http"
	"sync"

	"modular/modules/catalog"
	"modular/modules/customers"
	"modular/modules/orders/internal/pricing"
	"modular/modules/orders/internal/store"
	"modular/modules/payments"
	"modular/platform/events"
	"modular/platform/web"
)

// Event is the public contract of what happened to an order.
type Event struct {
	CustomerID string
	OrderID    string
	Type       string
}

type Module struct {
	mu        sync.Mutex // in a real database, this would be a transaction
	store     *store.Store
	catalog   *catalog.Module
	customers *customers.Module
	payments  *payments.Module
	bus       *events.Bus
}

func New(c *catalog.Module, cu *customers.Module, p *payments.Module, bus *events.Bus) *Module {
	return &Module{store: store.New(), catalog: c, customers: cu, payments: p, bus: bus}
}

type placeInput struct {
	CustomerID string `json:"customer_id"`
	Coupon     string `json:"coupon"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func (m *Module) place(in placeInput) (store.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// R04
	if len(in.Items) == 0 {
		return store.Order{}, web.Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	lines := make([]catalog.Line, 0, len(in.Items))
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return store.Order{}, web.Fail("validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
		lines = append(lines, catalog.Line{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	// R05
	if !m.customers.Exists(in.CustomerID) {
		return store.Order{}, web.Fail("customer_not_found", "customer does not exist")
	}
	// R06 and R07: the catalog reserves all or nothing and returns the current price
	reserved, err := m.catalog.Reserve(lines)
	if err != nil {
		return store.Order{}, err
	}

	order := store.Order{CustomerID: in.CustomerID, Status: "pending", Coupon: in.Coupon}
	for _, r := range reserved {
		order.Items = append(order.Items, store.Item{ProductID: r.ProductID, Quantity: r.Quantity, UnitPriceCents: r.UnitPriceCents})
		order.SubtotalCents += r.UnitPriceCents * r.Quantity
	}

	// R08: a refused coupon undoes the reservation (compensation)
	order.DiscountCents, err = pricing.Discount(in.Coupon, order.SubtotalCents, !m.store.HasActiveOrder(in.CustomerID))
	if err != nil {
		m.catalog.Release(lines)
		return store.Order{}, couponError(err)
	}
	// R09
	order.ShippingCents = pricing.Shipping(order.SubtotalCents - order.DiscountCents)
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	order = m.store.Insert(order)
	m.publish(order, "order_created") // R14
	return order, nil
}

func couponError(err error) error {
	if errors.Is(err, pricing.ErrInvalidCoupon) {
		return web.Fail("invalid_coupon", err.Error())
	}
	return web.Fail("coupon_not_eligible", err.Error())
}

func (m *Module) get(id string) (store.Order, error) {
	order, ok := m.store.Find(id)
	if !ok {
		return store.Order{}, web.Fail("not_found", "order not found")
	}
	return order, nil
}

func (m *Module) pay(id string) (store.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, err := m.get(id)
	if err != nil {
		return store.Order{}, err
	}
	// R11
	if order.Status != "pending" {
		return store.Order{}, web.Fail("invalid_transition", "only a pending order can be paid")
	}
	// R12
	if err := m.payments.Charge(order.TotalCents); err != nil {
		return store.Order{}, err
	}
	order.Status = "paid"
	m.store.Update(order)
	m.publish(order, "order_paid") // R14
	return order, nil
}

func (m *Module) cancel(id string) (store.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, err := m.get(id)
	if err != nil {
		return store.Order{}, err
	}
	// R11
	if order.Status == "cancelled" {
		return store.Order{}, web.Fail("invalid_transition", "order already cancelled")
	}
	// R13
	lines := make([]catalog.Line, len(order.Items))
	for i, item := range order.Items {
		lines[i] = catalog.Line{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	m.catalog.Release(lines)
	order.Status = "cancelled"
	m.store.Update(order)
	m.publish(order, "order_cancelled") // R14
	return order, nil
}

func (m *Module) publish(order store.Order, kind string) {
	m.bus.Publish(Event{CustomerID: order.CustomerID, OrderID: order.ID, Type: kind})
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in placeInput
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.place(in))
		}
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.get(r.PathValue("id")))
	})
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.pay(r.PathValue("id")))
	})
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.cancel(r.PathValue("id")))
	})
}
