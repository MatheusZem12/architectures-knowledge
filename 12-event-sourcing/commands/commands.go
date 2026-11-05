// Package commands decides. Each command follows the same three steps:
// replay the state from the events, check the rules, append new events.
package commands

import (
	"fmt"
	"strings"
	"sync"

	"eventsourcing/domain"
	"eventsourcing/events"
	"eventsourcing/eventstore"
	"eventsourcing/web"
)

const (
	bigOrderMinimum  = 50_000
	freeShippingFrom = 20_000
	shippingPrice    = 2_000
	maxChargeCents   = 1_000_000
)

type Handler struct {
	mu     sync.Mutex // a real event store would use optimistic concurrency per stream
	store  *eventstore.Store
	emails map[string]bool // a lookup that replaying one stream cannot answer
}

func New(store *eventstore.Store) *Handler {
	h := &Handler{store: store, emails: map[string]bool{}}
	store.Subscribe(func(record eventstore.Record) {
		if e, ok := record.Event.(events.CustomerRegistered); ok {
			h.emails[strings.ToLower(e.Email)] = true
		}
	})
	return h
}

func (h *Handler) newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, h.store.Len()+1)
}

func (h *Handler) CreateProduct(name string, priceCents, stock int) (string, error) {
	// R01
	if strings.TrimSpace(name) == "" || priceCents <= 0 || stock < 0 {
		return "", web.Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.newID("prod")
	return id, h.store.Append(eventstore.Entry{Stream: id, Event: events.ProductCreated{ProductID: id, Name: name, PriceCents: priceCents, Stock: stock}})
}

func (h *Handler) RegisterCustomer(name, email string) (string, error) {
	// R02
	at := strings.Index(email, "@")
	if strings.TrimSpace(name) == "" || at <= 0 || at == len(email)-1 {
		return "", web.Fail("validation", "customer needs a name and a valid email")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// R03
	if h.emails[strings.ToLower(email)] {
		return "", web.Fail("email_taken", "email already registered")
	}
	id := h.newID("cust")
	return id, h.store.Append(eventstore.Entry{Stream: id, Event: events.CustomerRegistered{CustomerID: id, Name: name, Email: email}})
}

type Line struct {
	ProductID string
	Quantity  int
}

func (h *Handler) PlaceOrder(customerID string, lines []Line, coupon string) (string, error) {
	// R04
	if len(lines) == 0 {
		return "", web.Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Quantity < 1 || line.Quantity > 10 || seen[line.ProductID] {
			return "", web.Fail("validation", "item with invalid quantity or repeated")
		}
		seen[line.ProductID] = true
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// R05
	customer := domain.ReplayCustomer(h.store.Load(customerID))
	if !customer.Exists {
		return "", web.Fail("customer_not_found", "customer does not exist")
	}

	id := h.newID("ord")
	placed := events.OrderPlaced{OrderID: id, CustomerID: customerID, Coupon: coupon}
	var entries []eventstore.Entry
	for _, line := range lines {
		// R05, R06 and R07: every product's state is replayed, and nothing is appended until the end
		product := domain.ReplayProduct(h.store.Load(line.ProductID))
		if !product.Exists {
			return "", web.Fail("product_not_found", "product does not exist")
		}
		if product.Stock < line.Quantity {
			return "", web.Fail("insufficient_stock", "insufficient stock")
		}
		placed.Items = append(placed.Items, events.OrderItem{ProductID: line.ProductID, Quantity: line.Quantity, UnitPriceCents: product.PriceCents})
		placed.SubtotalCents += product.PriceCents * line.Quantity
		entries = append(entries, eventstore.Entry{Stream: line.ProductID, Event: events.StockReserved{ProductID: line.ProductID, OrderID: id, Quantity: line.Quantity}})
	}

	discount, err := discountFor(coupon, placed.SubtotalCents, customer.ActiveOrders == 0)
	if err != nil {
		return "", err
	}
	placed.DiscountCents = discount
	placed.ShippingCents = shippingFor(placed.SubtotalCents - discount)
	// R10
	placed.TotalCents = placed.SubtotalCents - placed.DiscountCents + placed.ShippingCents

	entries = append(entries,
		eventstore.Entry{Stream: customerID, Event: events.CustomerOrderOpened{CustomerID: customerID, OrderID: id}},
		eventstore.Entry{Stream: id, Event: placed},
	)
	return id, h.store.Append(entries...)
}

func (h *Handler) PayOrder(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	order := domain.ReplayOrder(h.store.Load(id))
	if !order.Exists {
		return web.Fail("not_found", "order not found")
	}
	// R11
	if order.Status != "pending" {
		return web.Fail("invalid_transition", "only a pending order can be paid")
	}
	// R12
	if order.TotalCents > maxChargeCents {
		return web.Fail("payment_declined", "payment declined")
	}
	return h.store.Append(eventstore.Entry{Stream: id, Event: events.OrderPaid{OrderID: id, CustomerID: order.CustomerID}})
}

func (h *Handler) CancelOrder(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	order := domain.ReplayOrder(h.store.Load(id))
	if !order.Exists {
		return web.Fail("not_found", "order not found")
	}
	// R11
	if order.Status == "cancelled" {
		return web.Fail("invalid_transition", "order already cancelled")
	}
	// R13: nothing is undone. A new fact says the stock came back.
	entries := []eventstore.Entry{}
	for _, item := range order.Items {
		entries = append(entries, eventstore.Entry{Stream: item.ProductID, Event: events.StockReleased{ProductID: item.ProductID, OrderID: id, Quantity: item.Quantity}})
	}
	entries = append(entries,
		eventstore.Entry{Stream: order.CustomerID, Event: events.CustomerOrderClosed{CustomerID: order.CustomerID, OrderID: id}},
		eventstore.Entry{Stream: id, Event: events.OrderCancelled{OrderID: id, CustomerID: order.CustomerID}},
	)
	return h.store.Append(entries...)
}

// discountFor applies R08.
func discountFor(coupon string, subtotal int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, web.Fail("coupon_not_eligible", "coupon valid only on the first order")
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimum {
			return 0, web.Fail("coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, web.Fail("invalid_coupon", "invalid coupon")
	}
}

// shippingFor applies R09.
func shippingFor(afterDiscount int) int {
	if afterDiscount >= freeShippingFrom {
		return 0
	}
	return shippingPrice
}
