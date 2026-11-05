// Package getproduct is a simple read, so the slice is simple too. There is no layer to fill in.
package getproduct

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		var product store.Product
		err := db.Atomic(func() error {
			found, ok := db.Products[r.PathValue("id")]
			if !ok {
				return web.Fail("not_found", "product not found")
			}
			product = found
			return nil
		})
		web.Reply(w, 200)(product, err)
	})
}
