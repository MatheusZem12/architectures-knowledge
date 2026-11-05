// Package placeorder is the richest slice in the system, so it has the most code. Each slice is
// as complex as its own feature, and no more.
//
// Everything about pricing lives here, because nothing else needs it.
package placeorder

import (
	"net/http"

	"verticalslice/store"
	"verticalslice/web"
)

const (
	bigOrderMinimum  = 50_000
	freeShippingFrom = 20_000
	shippingPrice    = 2_000
)

type request struct {
	CustomerID string `json:"customer_id"`
	Coupon     string `json:"coupon"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func Register(mux *http.ServeMux, db *store.DB) {
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var in request
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(handle(db, in))
		}
	})
}

func handle(db *store.DB, in request) (store.Order, error) {
	// R04
	if len(in.Items) == 0 {
		return store.Order{}, web.Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return store.Order{}, web.Fail("validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
	}

	var order store.Order
	err := db.Atomic(func() error {
		// R05
		if _, ok := db.Customers[in.CustomerID]; !ok {
			return web.Fail("customer_not_found", "customer does not exist")
		}
		// R05, R06 and R07: check everything before changing any stock
		order = store.Order{CustomerID: in.CustomerID, Status: "pending", Coupon: in.Coupon}
		for _, item := range in.Items {
			product, ok := db.Products[item.ProductID]
			if !ok {
				return web.Fail("product_not_found", "product does not exist")
			}
			if product.Stock < item.Quantity {
				return web.Fail("insufficient_stock", "insufficient stock")
			}
			order.Items = append(order.Items, store.OrderItem{ProductID: product.ID, Quantity: item.Quantity, UnitPriceCents: product.PriceCents})
			order.SubtotalCents += product.PriceCents * item.Quantity
		}

		discount, err := discountFor(in.Coupon, order.SubtotalCents, !hasActiveOrder(db, in.CustomerID))
		if err != nil {
			return err
		}
		order.DiscountCents = discount
		// R09
		order.ShippingCents = shippingPrice
		if order.SubtotalCents-order.DiscountCents >= freeShippingFrom {
			order.ShippingCents = 0
		}
		// R10
		order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

		// R06: only save the stock once nothing else can fail
		for _, item := range order.Items {
			product := db.Products[item.ProductID]
			product.Stock -= item.Quantity
			db.Products[item.ProductID] = product
		}
		order.ID = db.NextID("ord")
		db.Orders[order.ID] = order
		db.Notify(order, "order_created") // R14
		return nil
	})
	return order, err
}

func hasActiveOrder(db *store.DB, customerID string) bool {
	for _, order := range db.Orders {
		if order.CustomerID == customerID && order.Status != "cancelled" {
			return true
		}
	}
	return false
}

// R08
func discountFor(coupon string, subtotal int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, web.Fail("coupon_not_eligible", "coupon valid only on the first order")
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimum {
			return 0, web.Fail("coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, web.Fail("invalid_coupon", "invalid coupon")
	}
}
