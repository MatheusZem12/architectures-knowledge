// Customer service: the enterprise service for customer records.
package main

import (
	"encoding/json"
	"strings"
	"sync"

	"soa/canonical"
)

var (
	db = canonical.NewDB()
	mu sync.Mutex // keeps the "check then create" of R03 free of races
)

func main() {
	canonical.Serve("customer", map[string]canonical.Handler{
		"CreateCustomer": createCustomer,
		"GetCustomer":    getCustomer,
	})
}

func createCustomer(payload json.RawMessage) (any, error) {
	var in canonical.Customer
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	// R02
	at := strings.Index(in.Email, "@")
	if strings.TrimSpace(in.Name) == "" || at <= 0 || at == len(in.Email)-1 {
		return nil, canonical.Fail(422, "validation", "customer needs a name and a valid email")
	}
	mu.Lock()
	defer mu.Unlock()
	// R03
	emailKey := strings.ToLower(in.Email)
	var owner string
	taken, err := db.Get("emails", emailKey, &owner)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, canonical.Fail(409, "email_taken", "email already registered")
	}
	if in.ID, err = db.NextID("cust"); err != nil {
		return nil, err
	}
	if err := db.Put("emails", emailKey, in.ID); err != nil {
		return nil, err
	}
	return in, db.Put("customers", in.ID, in)
}

func getCustomer(payload json.RawMessage) (any, error) {
	var in canonical.ByID
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	var customer canonical.Customer
	found, err := db.Get("customers", in.ID, &customer)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, canonical.Fail(404, "not_found", "customer not found")
	}
	return customer, nil
}
