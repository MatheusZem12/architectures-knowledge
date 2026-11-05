// Package payorder pays an order. The payment gateway lives here, because only this slice uses it.
package payorder

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

const maxChargeCents = 1_000_000

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("POST /orders/{id}/pay", func(w http.ResponseWriter, r *http.Request) {
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
		if found.Status != "pending" {
			return web.Fail("invalid_transition", "only a pending order can be paid")
		}
		// R12
		if found.TotalCents > maxChargeCents {
			return web.Fail("payment_declined", "payment declined")
		}
		found.Status = "paid"
		db.Orders[id] = found
		db.Notify(found, "order_paid") // R14
		order = found
		return nil
	})
	return order, err
}
