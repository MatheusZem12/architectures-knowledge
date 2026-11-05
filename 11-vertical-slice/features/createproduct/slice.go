// Package createproduct is one whole feature: request, rules, data access and response.
package createproduct

import (
	"net/http"
	"strings"

	"verticalslice/store"
	"verticalslice/web"
)

type request struct {
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in request
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(handle(db, in))
		}
	})
}

func handle(db *store.DB, in request) (store.Product, error) {
	// R01
	if strings.TrimSpace(in.Name) == "" || in.PriceCents <= 0 || in.Stock < 0 {
		return store.Product{}, web.Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	var product store.Product
	err := db.Atomic(func() error {
		product = store.Product{ID: db.NextID("prod"), Name: in.Name, PriceCents: in.PriceCents, Stock: in.Stock}
		db.Products[product.ID] = product
		return nil
	})
	return product, err
}
