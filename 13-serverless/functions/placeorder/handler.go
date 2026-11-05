// Package placeorder is one deployable function: stateless, and started on demand.
package placeorder

import (
	"serverless/cloud"
	"serverless/functions/shared"
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

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var in request
	if response, ok := shared.Decode(req, &in); !ok {
		return response
	}
	// R04
	if len(in.Items) == 0 {
		return shared.Fail(shared.Refuse(422, "validation", "order needs at least one item"))
	}
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return shared.Fail(shared.Refuse(422, "validation", "item with invalid quantity or repeated"))
		}
		seen[item.ProductID] = true
	}

	order := shared.Order{CustomerID: in.CustomerID, Status: "pending", Coupon: in.Coupon}
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		// R05
		var customer shared.Customer
		if !tx.Get("customer:"+in.CustomerID, &customer) {
			return shared.Refuse(422, "customer_not_found", "customer does not exist")
		}
		// R05, R06 and R07: a failure anywhere in this transaction discards every write in it
		for _, item := range in.Items {
			var product shared.Product
			if !tx.Get("product:"+item.ProductID, &product) {
				return shared.Refuse(422, "product_not_found", "product does not exist")
			}
			if product.Stock < item.Quantity {
				return shared.Refuse(409, "insufficient_stock", "insufficient stock")
			}
			product.Stock -= item.Quantity
			tx.Put("product:"+product.ID, product)
			order.Items = append(order.Items, shared.OrderItem{ProductID: product.ID, Quantity: item.Quantity, UnitPriceCents: product.PriceCents})
			order.SubtotalCents += product.PriceCents * item.Quantity
		}

		activeKey := "active-orders:" + in.CustomerID
		var active int
		tx.Get(activeKey, &active)
		discount, err := discountFor(in.Coupon, order.SubtotalCents, active == 0)
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

		order.ID = tx.NextID("ord")
		tx.Put("order:"+order.ID, order)
		tx.Put(activeKey, active+1)
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	env.Queue.Publish(shared.OrderEventsTopic, shared.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_created"}) // R14
	return shared.OK(201, order)
}

// R08
func discountFor(coupon string, subtotal int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, shared.Refuse(422, "coupon_not_eligible", "coupon valid only on the first order")
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimum {
			return 0, shared.Refuse(422, "coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, shared.Refuse(422, "invalid_coupon", "invalid coupon")
	}
}
