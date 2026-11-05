package kernel

import (
	"fmt"
	"net/http"
	"strings"
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

type Line struct {
	ProductID string
	Quantity  int
}

type Kernel struct {
	mu        sync.Mutex // in a real database, this would be a transaction
	products  map[string]Product
	customers map[string]Customer
	orders    map[string]Order
	seq       int

	mux       *http.ServeMux
	plugins   []string
	coupons   map[string]Coupon
	shipping  ShippingPolicy
	payment   PaymentProvider
	listeners []Listener
}

// New creates a kernel that already works with no plugins at all: no coupons, free shipping,
// and every payment declined. A plugin only ever adds to this.
func New() *Kernel {
	k := &Kernel{
		products:  map[string]Product{},
		customers: map[string]Customer{},
		orders:    map[string]Order{},
		mux:       http.NewServeMux(),
		coupons:   map[string]Coupon{},
		shipping:  func(int) int { return 0 },
		payment:   func(int) bool { return false },
	}
	k.routes()
	return k
}

func (k *Kernel) Load(plugin Plugin) {
	plugin.Install(&Registry{kernel: k})
	k.plugins = append(k.plugins, plugin.Name())
}

func (k *Kernel) Plugins() []string { return k.plugins }

func (k *Kernel) nextID(prefix string) string {
	k.seq++
	return fmt.Sprintf("%s-%d", prefix, k.seq)
}

func (k *Kernel) createProduct(p Product) (Product, error) {
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		return Product{}, Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	p.ID = k.nextID("prod")
	k.products[p.ID] = p
	return p, nil
}

func (k *Kernel) createCustomer(c Customer) (Customer, error) {
	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return Customer{}, Fail("validation", "customer needs a name and a valid email")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	// R03
	for _, existing := range k.customers {
		if strings.EqualFold(existing.Email, c.Email) {
			return Customer{}, Fail("email_taken", "email already registered")
		}
	}
	c.ID = k.nextID("cust")
	k.customers[c.ID] = c
	return c, nil
}

func (k *Kernel) placeOrder(customerID string, lines []Line, couponCode string) (Order, error) {
	// R04
	if len(lines) == 0 {
		return Order{}, Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Quantity < 1 || line.Quantity > 10 || seen[line.ProductID] {
			return Order{}, Fail("validation", "item with invalid quantity or repeated")
		}
		seen[line.ProductID] = true
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	// R05
	if _, ok := k.customers[customerID]; !ok {
		return Order{}, Fail("customer_not_found", "customer does not exist")
	}
	// R05, R06 and R07: check everything before changing any stock
	order := Order{CustomerID: customerID, Status: "pending", Coupon: couponCode}
	for _, line := range lines {
		product, ok := k.products[line.ProductID]
		if !ok {
			return Order{}, Fail("product_not_found", "product does not exist")
		}
		if product.Stock < line.Quantity {
			return Order{}, Fail("insufficient_stock", "insufficient stock")
		}
		order.Items = append(order.Items, OrderItem{product.ID, line.Quantity, product.PriceCents})
		order.SubtotalCents += product.PriceCents * line.Quantity
	}

	// R08: the kernel only knows that coupons exist. The plugins hold the rules.
	if couponCode != "" {
		coupon, ok := k.coupons[couponCode]
		if !ok {
			return Order{}, Fail("invalid_coupon", "invalid coupon")
		}
		discount, err := coupon.Discount(order.SubtotalCents, !k.hasActiveOrder(customerID))
		if err != nil {
			return Order{}, err
		}
		order.DiscountCents = discount
	}
	order.ShippingCents = k.shipping(order.SubtotalCents - order.DiscountCents)
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	// R06: only save the stock once nothing else can fail
	for _, item := range order.Items {
		product := k.products[item.ProductID]
		product.Stock -= item.Quantity
		k.products[item.ProductID] = product
	}
	order.ID = k.nextID("ord")
	k.orders[order.ID] = order
	k.emit(order, "order_created")
	return order, nil
}

func (k *Kernel) hasActiveOrder(customerID string) bool {
	for _, order := range k.orders {
		if order.CustomerID == customerID && order.Status != "cancelled" {
			return true
		}
	}
	return false
}

func (k *Kernel) findOrder(id string) (Order, error) {
	order, ok := k.orders[id]
	if !ok {
		return Order{}, Fail("not_found", "order not found")
	}
	return order, nil
}

func (k *Kernel) payOrder(id string) (Order, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	order, err := k.findOrder(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status != "pending" {
		return Order{}, Fail("invalid_transition", "only a pending order can be paid")
	}
	if !k.payment(order.TotalCents) {
		return Order{}, Fail("payment_declined", "payment declined")
	}
	order.Status = "paid"
	k.orders[id] = order
	k.emit(order, "order_paid")
	return order, nil
}

func (k *Kernel) cancelOrder(id string) (Order, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	order, err := k.findOrder(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status == "cancelled" {
		return Order{}, Fail("invalid_transition", "order already cancelled")
	}
	// R13
	for _, item := range order.Items {
		product := k.products[item.ProductID]
		product.Stock += item.Quantity
		k.products[item.ProductID] = product
	}
	order.Status = "cancelled"
	k.orders[id] = order
	k.emit(order, "order_cancelled")
	return order, nil
}

func (k *Kernel) emit(order Order, kind string) {
	for _, listener := range k.listeners {
		listener(OrderEvent{order.CustomerID, order.ID, kind})
	}
}
