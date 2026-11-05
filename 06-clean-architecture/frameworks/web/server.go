// Package web is the outermost ring: only here is it known that we use net/http.
// Switching framework rewrites this file and the line in main.go that starts the server.
package web

import (
	"encoding/json"
	"io"
	"net/http"

	"cleanarch/adapters/controllers"
)

func NewServer(c *controllers.Controller) http.Handler {
	mux := http.NewServeMux()
	route := func(pattern string, action func(controllers.Request) controllers.Response) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			response := action(controllers.Request{PathID: r.PathValue("id"), Body: body})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.Status)
			_ = json.NewEncoder(w).Encode(response.Body)
		})
	}
	route("GET /health", func(controllers.Request) controllers.Response {
		return controllers.Response{Status: 200, Body: map[string]string{"status": "ok"}}
	})
	route("POST /products", c.CreateProduct)
	route("GET /products/{id}", c.GetProduct)
	route("POST /customers", c.CreateCustomer)
	route("GET /customers/{id}", c.GetCustomer)
	route("GET /customers/{id}/notifications", c.ListNotifications)
	route("POST /orders", c.PlaceOrder)
	route("GET /orders/{id}", c.GetOrder)
	route("POST /orders/{id}/pay", c.PayOrder)
	route("POST /orders/{id}/cancel", c.CancelOrder)
	return mux
}
