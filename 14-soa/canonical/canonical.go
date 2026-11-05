// Package canonical is the enterprise's shared vocabulary: every service imports it.
//
// This is the sharpest difference from folder 07, where services shared nothing. Here, one
// agreed model avoids translations between services. The price: a change here touches everyone.
package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

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

// ByID is the payload of any operation that only needs an identifier.
type ByID struct {
	ID string `json:"id"`
}

type Charge struct {
	AmountCents int `json:"amount_cents"`
}

// Envelope is the one message format on the bus: an operation name and its payload.
type Envelope struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

// Fault is the one error format on the bus.
type Fault struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"error"`
}

func (f *Fault) Error() string { return f.Message }

func Fail(status int, code, message string) *Fault { return &Fault{status, code, message} }

var client = &http.Client{
	Timeout:   2 * time.Second,
	Transport: &http.Transport{MaxIdleConnsPerHost: 32},
}

// Handler implements one operation of a service.
type Handler func(payload json.RawMessage) (any, error)

// Serve exposes a service's operations. Services do not have routes of their own: only operations.
func Serve(name string, operations map[string]Handler) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { WriteJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /invoke", func(w http.ResponseWriter, r *http.Request) {
		var envelope Envelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			WriteFault(w, Fail(400, "invalid_json", "invalid JSON"))
			return
		}
		operation, ok := operations[envelope.Operation]
		if !ok {
			WriteFault(w, Fail(500, "internal", "unknown operation: "+envelope.Operation))
			return
		}
		result, err := operation(envelope.Payload)
		if err != nil {
			WriteFault(w, err)
			return
		}
		WriteJSON(w, 200, result)
	})
	port := os.Getenv("PORT")
	log.Println(name + " on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Decode reads an operation's payload.
func Decode(payload json.RawMessage, into any) error {
	if err := json.Unmarshal(payload, into); err != nil {
		return Fail(400, "invalid_json", "invalid JSON")
	}
	return nil
}

// Bus is how a service asks for something. It names an operation, and never a service.
type Bus struct {
	URL string
}

func NewBus() Bus { return Bus{URL: os.Getenv("ESB_URL")} }

func (b Bus) Call(operation string, payload, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return post(b.URL+"/invoke", Envelope{operation, raw}, out)
}

func post(url string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	res, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return Fail(503, "service_unavailable", "service unavailable")
	}
	defer res.Body.Close()
	return readReply(res, out)
}

func readReply(res *http.Response, out any) error {
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return Fail(503, "service_unavailable", "service unavailable")
	}
	if res.StatusCode >= 400 {
		fault := &Fault{Status: res.StatusCode}
		_ = json.Unmarshal(raw, fault)
		return fault
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// DB is the client of the shared database. Every service uses the same tables, directly.
type DB struct {
	URL string
}

func NewDB() DB { return DB{URL: os.Getenv("DATABASE_URL")} }

func (d DB) Get(table, id string, into any) (bool, error) {
	res, err := client.Get(fmt.Sprintf("%s/rows/%s/%s", d.URL, table, id))
	if err != nil {
		return false, Fail(503, "service_unavailable", "database unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		_, _ = io.Copy(io.Discard, res.Body)
		return false, nil
	}
	return true, readReply(res, into)
}

func (d DB) Put(table, id string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/rows/%s/%s", d.URL, table, id), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return Fail(503, "service_unavailable", "database unavailable")
	}
	defer res.Body.Close()
	return readReply(res, nil)
}

// List returns the rows of a table whose field has the given value.
func (d DB) List(table, field, value string, into any) error {
	res, err := client.Get(fmt.Sprintf("%s/rows/%s?%s=%s", d.URL, table, field, value))
	if err != nil {
		return Fail(503, "service_unavailable", "database unavailable")
	}
	defer res.Body.Close()
	return readReply(res, into)
}

func (d DB) NextID(prefix string) (string, error) {
	res, err := client.Post(fmt.Sprintf("%s/ids/%s", d.URL, prefix), "application/json", nil)
	if err != nil {
		return "", Fail(503, "service_unavailable", "database unavailable")
	}
	defer res.Body.Close()
	var out ByID
	err = readReply(res, &out)
	return out.ID, err
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteFault(w http.ResponseWriter, err error) {
	var fault *Fault
	if !errors.As(err, &fault) {
		fault = Fail(500, "internal", "internal error")
	}
	WriteJSON(w, fault.Status, fault)
}
