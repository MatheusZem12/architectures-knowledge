// Package write is the command side: it decides whether something may happen, and records it.
//
// It has no way to answer "show me this order". Its model holds only what decisions need: an
// order here does not even keep its coupon or subtotal, because no future decision uses them.
package write

import (
	"fmt"
	"strings"
	"sync"

	"cqrs/events"
)

// Publisher receives the facts. The write side does not know who is listening.
type Publisher interface {
	Publish(event any)
}

type PaymentGateway interface {
	Charge(amountCents int) bool
}

const maxChargeCents = 1_000_000

type FakeGateway struct{}

func (FakeGateway) Charge(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func fail(code, message string) *Error { return &Error{Code: code, Message: message} }

type product struct {
	priceCents int
	stock      int
}

type line struct {
	productID string
	quantity  int
}

type order struct {
	customerID string
	status     string
	lines      []line
	totalCents int
}

type Handler struct {
	mu        sync.Mutex // in a real database, this would be a transaction
	publisher Publisher
	gateway   PaymentGateway
	products  map[string]product
	customers map[string]bool
	emails    map[string]bool
	orders    map[string]order
	active    map[string]int // active orders per customer: answers R08 without scanning
	seq       int
}

func NewHandler(publisher Publisher, gateway PaymentGateway) *Handler {
	return &Handler{
		publisher: publisher,
		gateway:   gateway,
		products:  map[string]product{},
		customers: map[string]bool{},
		emails:    map[string]bool{},
		orders:    map[string]order{},
		active:    map[string]int{},
	}
}

func (h *Handler) nextID(prefix string) string {
	h.seq++
	return fmt.Sprintf("%s-%d", prefix, h.seq)
}

// A command is a request to change something. It returns an ID or a refusal, never a view.

type CreateProduct struct {
	Name       string
	PriceCents int
	Stock      int
}

func (h *Handler) CreateProduct(c CreateProduct) (string, error) {
	// R01
	if strings.TrimSpace(c.Name) == "" || c.PriceCents <= 0 || c.Stock < 0 {
		return "", fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.nextID("prod")
	h.products[id] = product{c.PriceCents, c.Stock}
	h.publisher.Publish(events.ProductCreated{ID: id, Name: c.Name, PriceCents: c.PriceCents, Stock: c.Stock})
	return id, nil
}

type RegisterCustomer struct {
	Name  string
	Email string
}

func (h *Handler) RegisterCustomer(c RegisterCustomer) (string, error) {
	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return "", fail("validation", "customer needs a name and a valid email")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// R03
	email := strings.ToLower(c.Email)
	if h.emails[email] {
		return "", fail("email_taken", "email already registered")
	}
	id := h.nextID("cust")
	h.emails[email] = true
	h.customers[id] = true
	h.publisher.Publish(events.CustomerRegistered{ID: id, Name: c.Name, Email: c.Email})
	return id, nil
}

type OrderLine struct {
	ProductID string
	Quantity  int
}

type PlaceOrder struct {
	CustomerID string
	Lines      []OrderLine
	Coupon     string
}

func (h *Handler) PlaceOrder(c PlaceOrder) (string, error) {
	// R04
	if len(c.Lines) == 0 {
		return "", fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, l := range c.Lines {
		if l.Quantity < 1 || l.Quantity > 10 || seen[l.ProductID] {
			return "", fail("validation", "item with invalid quantity or repeated")
		}
		seen[l.ProductID] = true
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// R05
	if !h.customers[c.CustomerID] {
		return "", fail("customer_not_found", "customer does not exist")
	}
	// R05, R06 and R07: check everything before changing any stock
	placed := events.OrderPlaced{CustomerID: c.CustomerID, Coupon: c.Coupon}
	lines := make([]line, len(c.Lines))
	for i, l := range c.Lines {
		p, ok := h.products[l.ProductID]
		if !ok {
			return "", fail("product_not_found", "product does not exist")
		}
		if p.stock < l.Quantity {
			return "", fail("insufficient_stock", "insufficient stock")
		}
		lines[i] = line{l.ProductID, l.Quantity}
		placed.Items = append(placed.Items, events.OrderItem{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceCents: p.priceCents})
		placed.SubtotalCents += p.priceCents * l.Quantity
	}

	discount, err := discountFor(c.Coupon, placed.SubtotalCents, h.active[c.CustomerID] == 0)
	if err != nil {
		return "", err
	}
	placed.DiscountCents = discount
	placed.ShippingCents = shippingFor(placed.SubtotalCents - discount)
	// R10
	placed.TotalCents = placed.SubtotalCents - placed.DiscountCents + placed.ShippingCents

	// R06: only save the stock once nothing else can fail
	for _, l := range lines {
		p := h.products[l.productID]
		p.stock -= l.quantity
		h.products[l.productID] = p
		h.publisher.Publish(events.StockChanged{ProductID: l.productID, Stock: p.stock})
	}
	placed.ID = h.nextID("ord")
	h.orders[placed.ID] = order{c.CustomerID, "pending", lines, placed.TotalCents}
	h.active[c.CustomerID]++
	h.publisher.Publish(placed)
	return placed.ID, nil
}

func (h *Handler) PayOrder(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	o, ok := h.orders[id]
	if !ok {
		return fail("not_found", "order not found")
	}
	// R11
	if o.status != "pending" {
		return fail("invalid_transition", "only a pending order can be paid")
	}
	if !h.gateway.Charge(o.totalCents) {
		return fail("payment_declined", "payment declined")
	}
	o.status = "paid"
	h.orders[id] = o
	h.publisher.Publish(events.OrderPaid{ID: id})
	return nil
}

func (h *Handler) CancelOrder(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	o, ok := h.orders[id]
	if !ok {
		return fail("not_found", "order not found")
	}
	// R11
	if o.status == "cancelled" {
		return fail("invalid_transition", "order already cancelled")
	}
	// R13
	for _, l := range o.lines {
		p := h.products[l.productID]
		p.stock += l.quantity
		h.products[l.productID] = p
		h.publisher.Publish(events.StockChanged{ProductID: l.productID, Stock: p.stock})
	}
	o.status = "cancelled"
	h.orders[id] = o
	h.active[o.customerID]--
	h.publisher.Publish(events.OrderCancelled{ID: id})
	return nil
}
