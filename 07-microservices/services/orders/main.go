// Orders service: coordinates the sale by talking to the other services.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

type Line struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

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
	Items      []Line `json:"items"`
	Coupon     string `json:"coupon"`
}

type Service struct {
	mu      sync.Mutex // simplification: a real system would use a transaction or optimistic locking
	items   map[string]Order
	seq     int
	clients *Clients
}

func NewService(clients *Clients) *Service {
	return &Service{items: map[string]Order{}, clients: clients}
}

func main() {
	s := NewService(NewClients(os.Getenv("CATALOG_URL"), os.Getenv("CUSTOMERS_URL"), os.Getenv("PAYMENTS_URL"), os.Getenv("NOTIFICATIONS_URL")))
	port := os.Getenv("PORT")
	log.Println("orders on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, s.Routes()))
}

func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in PlaceInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
			return
		}
		reply(w, 201)(s.Place(in))
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) { reply(w, 200)(s.Get(r.PathValue("id"))) })
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) { reply(w, 200)(s.Pay(r.PathValue("id"))) })
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) { reply(w, 200)(s.Cancel(r.PathValue("id"))) })
	return mux
}

// Place is a saga: a sequence of steps across services, with no shared transaction.
// If a step fails after the reservation, it must be undone explicitly.
func (s *Service) Place(in PlaceInput) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// R04
	if len(in.Items) == 0 {
		return Order{}, &RemoteError{422, "validation", "order needs at least one item"}
	}
	seen := map[string]bool{}
	for _, line := range in.Items {
		if line.Quantity < 1 || line.Quantity > 10 || seen[line.ProductID] {
			return Order{}, &RemoteError{422, "validation", "item with invalid quantity or repeated"}
		}
		seen[line.ProductID] = true
	}

	// R05: ask the customers service
	exists, err := s.clients.CustomerExists(in.CustomerID)
	if err != nil {
		return Order{}, err
	}
	if !exists {
		return Order{}, &RemoteError{422, "customer_not_found", "customer does not exist"}
	}

	// R06 and R07: saga step 1, the reservation in the catalog
	reserved, err := s.clients.Reserve(in.Items)
	if err != nil {
		return Order{}, err
	}

	order := Order{CustomerID: in.CustomerID, Status: "pending", Items: reserved, Coupon: in.Coupon}
	for _, item := range reserved {
		order.SubtotalCents += item.UnitPriceCents * item.Quantity
	}

	// R08: a refused coupon triggers the compensation, undoing step 1
	value, couponErr := discount(in.Coupon, order.SubtotalCents, !s.hasActiveOrder(in.CustomerID))
	if couponErr != nil {
		if err := s.clients.Release(in.Items); err != nil {
			// This is the worst case of distributed systems: the stock stays stuck until
			// someone reconciles it. A real system would use an outbox and retries.
			log.Printf("COMPENSATION FAILED, stock stuck: %v", err)
		}
		return Order{}, couponErr
	}
	order.DiscountCents = value
	// R09
	order.ShippingCents = shipping(order.SubtotalCents - order.DiscountCents)
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	s.seq++
	order.ID = fmt.Sprintf("ord-%d", s.seq)
	s.items[order.ID] = order
	s.clients.Notify(order.CustomerID, order.ID, "order_created") // R14
	return order, nil
}

func (s *Service) hasActiveOrder(customerID string) bool {
	for _, o := range s.items {
		if o.CustomerID == customerID && o.Status != "cancelled" {
			return true
		}
	}
	return false
}

func (s *Service) Get(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.find(id)
}

func (s *Service) find(id string) (Order, error) {
	order, ok := s.items[id]
	if !ok {
		return Order{}, &RemoteError{404, "not_found", "order not found"}
	}
	return order, nil
}

func (s *Service) Pay(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, err := s.find(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status != "pending" {
		return Order{}, &RemoteError{409, "invalid_transition", "only a pending order can be paid"}
	}
	// R12: the payments service decides
	if err := s.clients.Charge(order.TotalCents); err != nil {
		return Order{}, err
	}
	order.Status = "paid"
	s.items[id] = order
	s.clients.Notify(order.CustomerID, order.ID, "order_paid") // R14
	return order, nil
}

func (s *Service) Cancel(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, err := s.find(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status == "cancelled" {
		return Order{}, &RemoteError{409, "invalid_transition", "order already cancelled"}
	}
	// R13: if the catalog is down, the cancellation fails and the order does not change
	lines := make([]Line, len(order.Items))
	for i, item := range order.Items {
		lines[i] = Line{item.ProductID, item.Quantity}
	}
	if err := s.clients.Release(lines); err != nil {
		return Order{}, err
	}
	order.Status = "cancelled"
	s.items[id] = order
	s.clients.Notify(order.CustomerID, order.ID, "order_cancelled") // R14
	return order, nil
}

func reply(w http.ResponseWriter, status int) func(any, error) {
	return func(value any, err error) {
		if err == nil {
			writeJSON(w, status, value)
			return
		}
		var remote *RemoteError
		if errors.As(err, &remote) {
			writeJSON(w, remote.Status, map[string]string{"code": remote.Code, "error": remote.Message})
			return
		}
		if errors.Is(err, errUnavailable) {
			writeJSON(w, 503, map[string]string{"code": "service_unavailable", "error": "dependency unavailable"})
			return
		}
		writeJSON(w, 500, map[string]string{"code": "internal", "error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
