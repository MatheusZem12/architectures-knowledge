// Package api is the only place that knows both sides: POST goes to commands, GET goes to queries.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"cqrs/read"
	"cqrs/write"
)

func New(commands *write.Handler, views *read.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })

	// ---------- commands: change something, then read the result from the read side ----------

	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name       string `json:"name"`
			PriceCents int    `json:"price_cents"`
			Stock      int    `json:"stock"`
		}
		if !decode(w, r, &in) {
			return
		}
		id, err := commands.CreateProduct(write.CreateProduct{Name: in.Name, PriceCents: in.PriceCents, Stock: in.Stock})
		if err != nil {
			fail(w, err)
			return
		}
		view, _ := views.Product(id)
		respond(w, 201, view)
	})
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if !decode(w, r, &in) {
			return
		}
		id, err := commands.RegisterCustomer(write.RegisterCustomer{Name: in.Name, Email: in.Email})
		if err != nil {
			fail(w, err)
			return
		}
		view, _ := views.Customer(id)
		respond(w, 201, view)
	})
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			CustomerID string `json:"customer_id"`
			Coupon     string `json:"coupon"`
			Items      []struct {
				ProductID string `json:"product_id"`
				Quantity  int    `json:"quantity"`
			} `json:"items"`
		}
		if !decode(w, r, &in) {
			return
		}
		command := write.PlaceOrder{CustomerID: in.CustomerID, Coupon: in.Coupon}
		for _, item := range in.Items {
			command.Lines = append(command.Lines, write.OrderLine{ProductID: item.ProductID, Quantity: item.Quantity})
		}
		id, err := commands.PlaceOrder(command)
		if err != nil {
			fail(w, err)
			return
		}
		view, _ := views.Order(id)
		respond(w, 201, view)
	})
	transition := func(command func(string) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if err := command(id); err != nil {
				fail(w, err)
				return
			}
			view, _ := views.Order(id)
			respond(w, 200, view)
		}
	}
	mux.HandleFunc("POST /orders/{id}/pay", transition(commands.PayOrder))
	mux.HandleFunc("POST /orders/{id}/cancel", transition(commands.CancelOrder))

	// ---------- queries: never change anything ----------

	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Product(r.PathValue("id"))
		found(w, view, ok, "product not found")
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Customer(r.PathValue("id"))
		found(w, view, ok, "customer not found")
	})
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, views.Notifications(r.PathValue("id")))
	})
	// An extra route, beyond the SPEC: it shows a view that only exists because reading is separate.
	mux.HandleFunc("GET /customers/{id}/summary", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Summary(r.PathValue("id"))
		found(w, view, ok, "customer not found")
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Order(r.PathValue("id"))
		found(w, view, ok, "order not found")
	})
	return mux
}

func found(w http.ResponseWriter, view any, ok bool, message string) {
	if !ok {
		respond(w, 404, map[string]string{"code": "not_found", "error": message})
		return
	}
	respond(w, 200, view)
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		respond(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return false
	}
	return true
}

func fail(w http.ResponseWriter, err error) {
	var refusal *write.Error
	if !errors.As(err, &refusal) {
		respond(w, 500, map[string]string{"code": "internal", "error": "internal error"})
		return
	}
	respond(w, statusFor(refusal.Code), map[string]string{"code": refusal.Code, "error": refusal.Message})
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
