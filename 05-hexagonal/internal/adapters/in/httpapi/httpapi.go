// Package httpapi is an inbound adapter: it translates HTTP into calls to the ports.Shop port.
//
// JSON lives here. The DTOs below keep the domain free of tags and format details.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"hexagonal/internal/core/domain"
	"hexagonal/internal/core/ports"
)

type productDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type customerDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type itemDTO struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type orderDTO struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	Status        string    `json:"status"`
	Items         []itemDTO `json:"items"`
	Coupon        string    `json:"coupon"`
	SubtotalCents int       `json:"subtotal_cents"`
	DiscountCents int       `json:"discount_cents"`
	ShippingCents int       `json:"shipping_cents"`
	TotalCents    int       `json:"total_cents"`
}

type notificationDTO struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type placeOrderDTO struct {
	CustomerID string    `json:"customer_id"`
	Items      []itemDTO `json:"items"`
	Coupon     string    `json:"coupon"`
}

func New(shop ports.Shop) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		respond(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in productDTO
		if decode(w, r, &in) {
			p, err := shop.CreateProduct(in.Name, in.PriceCents, in.Stock)
			reply(w, 201, toProduct(p), err)
		}
	})
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := shop.GetProduct(r.PathValue("id"))
		reply(w, 200, toProduct(p), err)
	})
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in customerDTO
		if decode(w, r, &in) {
			c, err := shop.CreateCustomer(in.Name, in.Email)
			reply(w, 201, customerDTO(c), err)
		}
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, err := shop.GetCustomer(r.PathValue("id"))
		reply(w, 200, customerDTO(c), err)
	})
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		out := []notificationDTO{}
		for _, n := range shop.Notifications(r.PathValue("id")) {
			out = append(out, notificationDTO(n))
		}
		respond(w, 200, out)
	})
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in placeOrderDTO
		if decode(w, r, &in) {
			lines := make([]domain.Line, len(in.Items))
			for i, item := range in.Items {
				lines[i] = domain.Line{ProductID: item.ProductID, Quantity: item.Quantity}
			}
			o, err := shop.PlaceOrder(in.CustomerID, lines, in.Coupon)
			reply(w, 201, toOrder(o), err)
		}
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		o, err := shop.GetOrder(r.PathValue("id"))
		reply(w, 200, toOrder(o), err)
	})
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		o, err := shop.PayOrder(r.PathValue("id"))
		reply(w, 200, toOrder(o), err)
	})
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		o, err := shop.CancelOrder(r.PathValue("id"))
		reply(w, 200, toOrder(o), err)
	})
	return mux
}

func toProduct(p domain.Product) productDTO {
	return productDTO{p.ID, p.Name, p.PriceCents, p.Stock}
}

func toOrder(o domain.Order) orderDTO {
	items := make([]itemDTO, len(o.Items))
	for i, item := range o.Items {
		items[i] = itemDTO(item)
	}
	return orderDTO{o.ID, o.CustomerID, string(o.Status), items, o.Coupon, o.SubtotalCents, o.DiscountCents, o.ShippingCents, o.TotalCents}
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		respond(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return false
	}
	return true
}

func reply(w http.ResponseWriter, status int, value any, err error) {
	if err == nil {
		respond(w, status, value)
		return
	}
	var businessErr *domain.Error
	if errors.As(err, &businessErr) {
		respond(w, statusFor(businessErr.Code), map[string]string{"code": string(businessErr.Code), "error": businessErr.Message})
		return
	}
	respond(w, 500, map[string]string{"code": "internal", "error": "internal error"})
}

func statusFor(code domain.Code) int {
	switch code {
	case domain.CodeNotFound:
		return 404
	case domain.CodePaymentDeclined:
		return 402
	case domain.CodeEmailTaken, domain.CodeInsufficientStock, domain.CodeInvalidTransition:
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
