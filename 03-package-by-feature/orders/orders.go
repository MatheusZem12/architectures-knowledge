// Package orders gathers everything about orders.
//
// This is where the limit of this organization shows. Look at the imports: orders knows the
// structs and concrete stores of products, customers and notifications, and changes stock
// directly in products.Store. The folder is cohesive, but nothing prevents this coupling.
package orders

import (
	"fmt"
	"net/http"
	"sync"

	"byfeature/customers"
	"byfeature/notifications"
	"byfeature/payments"
	"byfeature/products"
	"byfeature/web"
)

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

type PlaceInput struct {
	CustomerID string `json:"customer_id"`
	Coupon     string `json:"coupon"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

type Service struct {
	mu            sync.Mutex // in a real database, this would be a transaction
	items         map[string]Order
	seq           int
	products      *products.Store
	customers     *customers.Store
	notifications *notifications.Store
}

func NewService(p *products.Store, c *customers.Store, n *notifications.Store) *Service {
	return &Service{items: map[string]Order{}, products: p, customers: c, notifications: n}
}

func (s *Service) Place(in PlaceInput) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// R04
	if len(in.Items) == 0 {
		return Order{}, web.Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return Order{}, web.Fail("validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
	}
	// R05
	if _, ok := s.customers.Find(in.CustomerID); !ok {
		return Order{}, web.Fail("customer_not_found", "customer does not exist")
	}

	// R05, R06 and R07: check everything before changing any stock
	order := Order{CustomerID: in.CustomerID, Status: "pending", Coupon: in.Coupon}
	reserved := make([]products.Product, 0, len(in.Items))
	for _, item := range in.Items {
		product, ok := s.products.Find(item.ProductID)
		if !ok {
			return Order{}, web.Fail("product_not_found", "product does not exist")
		}
		if product.Stock < item.Quantity {
			return Order{}, web.Fail("insufficient_stock", "insufficient stock")
		}
		order.Items = append(order.Items, Item{product.ID, item.Quantity, product.PriceCents})
		order.SubtotalCents += product.PriceCents * item.Quantity
		product.Stock -= item.Quantity
		reserved = append(reserved, product)
	}

	discount, err := s.discountFor(in.Coupon, in.CustomerID, order.SubtotalCents)
	if err != nil {
		return Order{}, err
	}
	order.DiscountCents = discount
	// R09
	order.ShippingCents = 2_000
	if order.SubtotalCents-order.DiscountCents >= 20_000 {
		order.ShippingCents = 0
	}
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	// R06: only save the stock once nothing else can fail
	for _, product := range reserved {
		s.products.Save(product)
	}
	s.seq++
	order.ID = fmt.Sprintf("ord-%d", s.seq)
	s.items[order.ID] = order
	s.notify(order, "order_created") // R14
	return order, nil
}

// R08
func (s *Service) discountFor(coupon, customerID string, subtotal int) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		for _, previous := range s.items {
			if previous.CustomerID == customerID && previous.Status != "cancelled" {
				return 0, web.Fail("coupon_not_eligible", "coupon valid only on the first order")
			}
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < 50_000 {
			return 0, web.Fail("coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, web.Fail("invalid_coupon", "invalid coupon")
	}
}

func (s *Service) Get(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.get(id)
}

func (s *Service) get(id string) (Order, error) {
	order, ok := s.items[id]
	if !ok {
		return Order{}, web.Fail("not_found", "order not found")
	}
	return order, nil
}

func (s *Service) Pay(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, err := s.get(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status != "pending" {
		return Order{}, web.Fail("invalid_transition", "only a pending order can be paid")
	}
	// R12
	if !payments.Charge(order.TotalCents) {
		return Order{}, web.Fail("payment_declined", "payment declined")
	}
	order.Status = "paid"
	s.items[id] = order
	s.notify(order, "order_paid") // R14
	return order, nil
}

func (s *Service) Cancel(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, err := s.get(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status == "cancelled" {
		return Order{}, web.Fail("invalid_transition", "order already cancelled")
	}
	// R13
	for _, item := range order.Items {
		if product, ok := s.products.Find(item.ProductID); ok {
			product.Stock += item.Quantity
			s.products.Save(product)
		}
	}
	order.Status = "cancelled"
	s.items[id] = order
	s.notify(order, "order_cancelled") // R14
	return order, nil
}

func (s *Service) notify(order Order, kind string) {
	s.notifications.Add(notifications.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: kind})
}

func Register(mux *http.ServeMux, s *Service) {
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in PlaceInput
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(s.Place(in))
		}
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(s.Get(r.PathValue("id")))
	})
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(s.Pay(r.PathValue("id")))
	})
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(s.Cancel(r.PathValue("id")))
	})
}
