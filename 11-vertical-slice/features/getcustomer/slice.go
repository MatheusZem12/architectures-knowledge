// Package getcustomer reads a customer.
package getcustomer

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		var customer store.Customer
		err := db.Atomic(func() error {
			found, ok := db.Customers[r.PathValue("id")]
			if !ok {
				return web.Fail("not_found", "customer not found")
			}
			customer = found
			return nil
		})
		web.Reply(w, 200)(customer, err)
	})
}
