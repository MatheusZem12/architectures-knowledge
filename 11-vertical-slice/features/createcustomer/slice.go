// Package createcustomer registers a customer.
package createcustomer

import (
	"net/http"
	"strings"

	"verticalslice/store"
	"verticalslice/web"
)

type request struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in request
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(handle(db, in))
		}
	})
}

func handle(db *store.DB, in request) (store.Customer, error) {
	// R02
	at := strings.Index(in.Email, "@")
	if strings.TrimSpace(in.Name) == "" || at <= 0 || at == len(in.Email)-1 {
		return store.Customer{}, web.Fail("validation", "customer needs a name and a valid email")
	}
	var customer store.Customer
	err := db.Atomic(func() error {
		// R03
		for _, existing := range db.Customers {
			if strings.EqualFold(existing.Email, in.Email) {
				return web.Fail("email_taken", "email already registered")
			}
		}
		customer = store.Customer{ID: db.NextID("cust"), Name: in.Name, Email: in.Email}
		db.Customers[customer.ID] = customer
		return nil
	})
	return customer, err
}
