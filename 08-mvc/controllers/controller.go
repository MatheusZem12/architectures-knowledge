// Package controllers is the C in MVC: it receives the request, asks the model, picks the view.
//
// Controllers stay thin on purpose. If a business decision shows up here, it belongs in the model.
package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"mvc/models"
	"mvc/views"
)

type Controller struct {
	db *models.DB
}

func New(db *models.DB) *Controller { return &Controller{db: db} }

func (c *Controller) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { views.Health(w) })
	mux.HandleFunc("POST /products", c.createProduct)
	mux.HandleFunc("GET /products/{id}", c.showProduct)
	mux.HandleFunc("POST /customers", c.createCustomer)
	mux.HandleFunc("GET /customers/{id}", c.showCustomer)
	mux.HandleFunc("GET /customers/{id}/notifications", c.listNotifications)
	mux.HandleFunc("POST /orders", c.createOrder)
	mux.HandleFunc("GET /orders/{id}", c.showOrder)
	mux.HandleFunc("POST /orders/{id}/pay", c.payOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", c.cancelOrder)
	return mux
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		views.Error(w, r, 400, "invalid_json", "invalid JSON")
		return false
	}
	return true
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	var businessErr *models.Error
	if !errors.As(err, &businessErr) {
		views.Error(w, r, 500, "internal", "internal error")
		return
	}
	views.Error(w, r, statusFor(businessErr.Code), businessErr.Code, businessErr.Message)
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
