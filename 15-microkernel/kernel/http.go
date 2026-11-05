package kernel

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// Fail is exported so that plugins can refuse things in the kernel's language.
func Fail(code, message string) *Error { return &Error{Code: code, Message: message} }

func (k *Kernel) Handler() http.Handler { return k.mux }

func (k *Kernel) routes() {
	k.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { JSON(w, 200, map[string]string{"status": "ok"}) })
	// An extra route, beyond the SPEC: shows what is installed right now.
	k.mux.HandleFunc("GET /plugins", func(w http.ResponseWriter, _ *http.Request) { JSON(w, 200, k.plugins) })
	k.mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in Product
		if decode(w, r, &in) {
			reply(w, 201)(k.createProduct(in))
		}
	})
	k.mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		defer k.mu.Unlock()
		product, ok := k.products[r.PathValue("id")]
		if !ok {
			reply(w, 200)(nil, Fail("not_found", "product not found"))
			return
		}
		JSON(w, 200, product)
	})
	k.mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in Customer
		if decode(w, r, &in) {
			reply(w, 201)(k.createCustomer(in))
		}
	})
	k.mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		defer k.mu.Unlock()
		customer, ok := k.customers[r.PathValue("id")]
		if !ok {
			reply(w, 200)(nil, Fail("not_found", "customer not found"))
			return
		}
		JSON(w, 200, customer)
	})
	k.mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
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
		lines := make([]Line, len(in.Items))
		for i, item := range in.Items {
			lines[i] = Line{item.ProductID, item.Quantity}
		}
		reply(w, 201)(k.placeOrder(in.CustomerID, lines, in.Coupon))
	})
	k.mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		defer k.mu.Unlock()
		reply(w, 200)(k.findOrder(r.PathValue("id")))
	})
	k.mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200)(k.payOrder(r.PathValue("id")))
	})
	k.mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200)(k.cancelOrder(r.PathValue("id")))
	})
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		JSON(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return false
	}
	return true
}

// JSON is exported so that a plugin's own routes answer in the same way.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func reply(w http.ResponseWriter, status int) func(any, error) {
	return func(value any, err error) {
		if err == nil {
			JSON(w, status, value)
			return
		}
		var businessErr *Error
		if errors.As(err, &businessErr) {
			JSON(w, statusFor(businessErr.Code), map[string]string{"code": businessErr.Code, "error": businessErr.Message})
			return
		}
		JSON(w, 500, map[string]string{"code": "internal", "error": "internal error"})
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
