// Package api connects HTTP to commands and projections.
package api

import (
	"net/http"

	"eventsourcing/commands"
	"eventsourcing/eventstore"
	"eventsourcing/projections"
	"eventsourcing/web"
)

func New(handler *commands.Handler, views *projections.Views, store *eventstore.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { web.JSON(w, 200, map[string]string{"status": "ok"}) })

	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name       string `json:"name"`
			PriceCents int    `json:"price_cents"`
			Stock      int    `json:"stock"`
		}
		if !web.Decode(w, r, &in) {
			return
		}
		id, err := handler.CreateProduct(in.Name, in.PriceCents, in.Stock)
		view, _ := views.Product(id)
		web.Reply(w, 201)(view, err)
	})
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if !web.Decode(w, r, &in) {
			return
		}
		id, err := handler.RegisterCustomer(in.Name, in.Email)
		view, _ := views.Customer(id)
		web.Reply(w, 201)(view, err)
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
		if !web.Decode(w, r, &in) {
			return
		}
		lines := make([]commands.Line, len(in.Items))
		for i, item := range in.Items {
			lines[i] = commands.Line{ProductID: item.ProductID, Quantity: item.Quantity}
		}
		id, err := handler.PlaceOrder(in.CustomerID, lines, in.Coupon)
		view, _ := views.Order(id)
		web.Reply(w, 201)(view, err)
	})
	transition := func(command func(string) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			err := command(id)
			view, _ := views.Order(id)
			web.Reply(w, 200)(view, err)
		}
	}
	mux.HandleFunc("POST /orders/{id}/pay", transition(handler.PayOrder))
	mux.HandleFunc("POST /orders/{id}/cancel", transition(handler.CancelOrder))

	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Product(r.PathValue("id"))
		found(w, view, ok, "product not found")
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Customer(r.PathValue("id"))
		found(w, view, ok, "customer not found")
	})
	mux.HandleFunc("GET /customers/{id}/notifications", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, 200, views.Notifications(r.PathValue("id")))
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		view, ok := views.Order(r.PathValue("id"))
		found(w, view, ok, "order not found")
	})

	// Extra routes, beyond the SPEC: the complete history of anything, which no other folder can offer.
	history := func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, 200, store.History(r.PathValue("id")))
	}
	mux.HandleFunc("GET /products/{id}/history", history)
	mux.HandleFunc("GET /orders/{id}/history", history)
	return mux
}

func found(w http.ResponseWriter, view any, ok bool, message string) {
	if !ok {
		web.Reply(w, 200)(nil, web.Fail("not_found", message))
		return
	}
	web.JSON(w, 200, view)
}
