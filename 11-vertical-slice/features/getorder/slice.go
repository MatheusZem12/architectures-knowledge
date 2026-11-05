// Package getorder reads an order.
package getorder

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		var order store.Order
		err := db.Atomic(func() error {
			found, ok := db.Orders[r.PathValue("id")]
			if !ok {
				return web.Fail("not_found", "order not found")
			}
			order = found
			return nil
		})
		web.Reply(w, 200)(order, err)
	})
}
