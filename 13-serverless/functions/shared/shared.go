// Package shared is what functions have in common, the role a "Lambda layer" plays in AWS.
// It holds types and helpers only. Like the functions, it holds no state.
package shared

import (
	"encoding/json"

	"serverless/cloud"
)

const OrderEventsTopic = "order-events"

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

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Refusal struct {
	Status  int
	Code    string
	Message string
}

func (r *Refusal) Error() string { return r.Message }

func Refuse(status int, code, message string) *Refusal {
	return &Refusal{status, code, message}
}

func OK(status int, body any) cloud.Response { return cloud.Response{Status: status, Body: body} }

// Fail turns an error into a response.
func Fail(err error) cloud.Response {
	refusal, ok := err.(*Refusal)
	if !ok {
		return cloud.Response{Status: 500, Body: map[string]string{"code": "internal", "error": "internal error"}}
	}
	return cloud.Response{Status: refusal.Status, Body: map[string]string{"code": refusal.Code, "error": refusal.Message}}
}

// Decode reads the request body. The second result is false if the JSON is malformed.
func Decode(req cloud.Request, into any) (cloud.Response, bool) {
	if err := json.Unmarshal(req.Body, into); err != nil {
		return Fail(Refuse(400, "invalid_json", "invalid JSON")), false
	}
	return cloud.Response{}, true
}
