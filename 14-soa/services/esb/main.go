// The Enterprise Service Bus: every call between services goes through here.
//
// It holds what folder 07's gateway refused to hold: knowledge about the system. It maps REST to
// operations, knows which service provides each operation, validates messages and logs traffic.
// That centralization is both the appeal of SOA and its biggest risk.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"soa/canonical"
)

// providers is the routing table: which service answers each operation. Moving an operation to
// another service changes this line, and none of the callers.
var providers = map[string]string{
	"CreateProduct":     "SALES_URL",
	"GetProduct":        "SALES_URL",
	"PlaceOrder":        "SALES_URL",
	"GetOrder":          "SALES_URL",
	"PayOrder":          "SALES_URL",
	"CancelOrder":       "SALES_URL",
	"CreateCustomer":    "CUSTOMER_URL",
	"GetCustomer":       "CUSTOMER_URL",
	"ChargePayment":     "BILLING_URL",
	"SendNotification":  "BILLING_URL",
	"ListNotifications": "BILLING_URL",
}

// restRoutes translates the public REST API into canonical operations.
var restRoutes = []struct {
	pattern   string
	operation string
	success   int
}{
	{"POST /products", "CreateProduct", 201},
	{"GET /products/{id}", "GetProduct", 200},
	{"POST /customers", "CreateCustomer", 201},
	{"GET /customers/{id}", "GetCustomer", 200},
	{"GET /customers/{id}/notifications", "ListNotifications", 200},
	{"POST /orders", "PlaceOrder", 201},
	{"GET /orders/{id}", "GetOrder", 200},
	{"POST /orders/{id}/pay", "PayOrder", 200},
	{"POST /orders/{id}/cancel", "CancelOrder", 200},
}

var client = &http.Client{
	Timeout:   5 * time.Second,
	Transport: &http.Transport{MaxIdleConnsPerHost: 32},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /invoke", invoke)
	for _, route := range restRoutes {
		mux.HandleFunc(route.pattern, func(w http.ResponseWriter, r *http.Request) {
			translate(w, r, route.operation, route.success)
		})
	}
	port := os.Getenv("PORT")
	log.Println("esb on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// translate turns a REST request into a canonical message. The services never see REST.
func translate(w http.ResponseWriter, r *http.Request, operation string, success int) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		canonical.WriteFault(w, canonical.Fail(400, "invalid_json", "invalid JSON"))
		return
	}
	if id := r.PathValue("id"); id != "" {
		payload, _ = json.Marshal(canonical.ByID{ID: id})
	}
	// The bus validates messages, so services do not get what they could not read.
	if !json.Valid(payload) {
		canonical.WriteFault(w, canonical.Fail(400, "invalid_json", "invalid JSON"))
		return
	}
	route(w, canonical.Envelope{Operation: operation, Payload: payload}, success)
}

// invoke handles calls between services.
func invoke(w http.ResponseWriter, r *http.Request) {
	var envelope canonical.Envelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		canonical.WriteFault(w, canonical.Fail(400, "invalid_json", "invalid JSON"))
		return
	}
	route(w, envelope, 200)
}

func route(w http.ResponseWriter, envelope canonical.Envelope, success int) {
	address := os.Getenv(providers[envelope.Operation])
	if address == "" {
		canonical.WriteFault(w, canonical.Fail(500, "internal", "no provider for "+envelope.Operation))
		return
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		canonical.WriteFault(w, err)
		return
	}
	res, err := client.Post(address+"/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		canonical.WriteFault(w, canonical.Fail(503, "service_unavailable", "service unavailable"))
		return
	}
	defer res.Body.Close()
	status := res.StatusCode
	if status == 200 {
		status = success
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.Copy(w, res.Body)
}

func health(w http.ResponseWriter, _ *http.Request) {
	var down []string
	for _, name := range []string{"DATABASE_URL", "CUSTOMER_URL", "SALES_URL", "BILLING_URL"} {
		res, err := client.Get(os.Getenv(name) + "/health")
		if err != nil {
			down = append(down, name)
			continue
		}
		res.Body.Close()
	}
	if len(down) > 0 {
		canonical.WriteFault(w, canonical.Fail(503, "service_unavailable", "down: "+strings.Join(down, ", ")))
		return
	}
	canonical.WriteJSON(w, 200, map[string]string{"status": "ok"})
}
