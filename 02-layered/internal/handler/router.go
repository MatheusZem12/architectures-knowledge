// Package handler is the presentation layer: it translates HTTP to and from service calls.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"layered/internal/model"
	"layered/internal/service"
)

type Router struct {
	products      *service.ProductService
	customers     *service.CustomerService
	orders        *service.OrderService
	notifications *service.NotificationService
}

func NewRouter(
	products *service.ProductService,
	customers *service.CustomerService,
	orders *service.OrderService,
	notifications *service.NotificationService,
) http.Handler {
	r := &Router{products, customers, orders, notifications}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /products", r.createProduct)
	mux.HandleFunc("GET /products/{id}", r.getProduct)
	mux.HandleFunc("POST /customers", r.createCustomer)
	mux.HandleFunc("GET /customers/{id}", r.getCustomer)
	mux.HandleFunc("GET /customers/{id}/notifications", r.listNotifications)
	mux.HandleFunc("POST /orders", r.placeOrder)
	mux.HandleFunc("GET /orders/{id}", r.getOrder)
	mux.HandleFunc("POST /orders/{id}/pay", r.payOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", r.cancelOrder)
	return mux
}

func (r *Router) createProduct(w http.ResponseWriter, req *http.Request) {
	var in model.Product
	if !decode(w, req, &in) {
		return
	}
	reply(w, 201)(r.products.Create(in))
}

func (r *Router) getProduct(w http.ResponseWriter, req *http.Request) {
	reply(w, 200)(r.products.Get(req.PathValue("id")))
}

func (r *Router) createCustomer(w http.ResponseWriter, req *http.Request) {
	var in model.Customer
	if !decode(w, req, &in) {
		return
	}
	reply(w, 201)(r.customers.Create(in))
}

func (r *Router) getCustomer(w http.ResponseWriter, req *http.Request) {
	reply(w, 200)(r.customers.Get(req.PathValue("id")))
}

func (r *Router) listNotifications(w http.ResponseWriter, req *http.Request) {
	respond(w, 200, r.notifications.ListByCustomer(req.PathValue("id")))
}

func (r *Router) placeOrder(w http.ResponseWriter, req *http.Request) {
	var in service.PlaceOrderInput
	if !decode(w, req, &in) {
		return
	}
	reply(w, 201)(r.orders.Place(in))
}

func (r *Router) getOrder(w http.ResponseWriter, req *http.Request) {
	reply(w, 200)(r.orders.Get(req.PathValue("id")))
}

func (r *Router) payOrder(w http.ResponseWriter, req *http.Request) {
	reply(w, 200)(r.orders.Pay(req.PathValue("id")))
}

func (r *Router) cancelOrder(w http.ResponseWriter, req *http.Request) {
	reply(w, 200)(r.orders.Cancel(req.PathValue("id")))
}

// ---------- helpers ----------

func decode(w http.ResponseWriter, req *http.Request, into any) bool {
	if err := json.NewDecoder(req.Body).Decode(into); err != nil {
		respond(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return false
	}
	return true
}

// reply receives a service's (value, error) and writes the right response.
func reply(w http.ResponseWriter, status int) func(any, error) {
	return func(value any, err error) {
		if err == nil {
			respond(w, status, value)
			return
		}
		var businessErr *service.Error
		if errors.As(err, &businessErr) {
			respond(w, statusFor(businessErr.Code), map[string]string{"code": businessErr.Code, "error": businessErr.Message})
			return
		}
		respond(w, 500, map[string]string{"code": "internal", "error": "internal error"})
	}
}

func statusFor(code string) int {
	switch code {
	case "not_found":
		return 404
	case "payment_declined":
		return 402
	case "email_taken", "insufficient_stock", "invalid_transition":
		return 409
	default:
		return 422
	}
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
