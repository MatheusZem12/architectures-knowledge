// Package cancelorder cancels an order and returns its stock.
package cancelorder

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(handle(db, r.PathValue("id")))
	})
}

func handle(db *store.DB, id string) (store.Order, error) {
	var order store.Order
	err := db.Atomic(func() error {
		found, ok := db.Orders[id]
		if !ok {
			return web.Fail("not_found", "order not found")
		}
		// R11
		if found.Status == "cancelled" {
			return web.Fail("invalid_transition", "order already cancelled")
		}
		// R13
		for _, item := range found.Items {
			product := db.Products[item.ProductID]
			product.Stock += item.Quantity
			db.Products[item.ProductID] = product
		}
		found.Status = "cancelled"
		db.Orders[id] = found
		db.Notify(found, "order_cancelled") // R14
		order = found
		return nil
	})
	return order, err
}
