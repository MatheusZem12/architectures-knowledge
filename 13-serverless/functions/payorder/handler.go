// Package payorder is one deployable function.
package payorder

import (
	"serverless/cloud"
	"serverless/functions/shared"
)

const maxChargeCents = 1_000_000

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var order shared.Order
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		if !tx.Get("order:"+req.PathID, &order) {
			return shared.Refuse(404, "not_found", "order not found")
		}
		// R11
		if order.Status != "pending" {
			return shared.Refuse(409, "invalid_transition", "only a pending order can be paid")
		}
		// R12
		if order.TotalCents > maxChargeCents {
			return shared.Refuse(402, "payment_declined", "payment declined")
		}
		order.Status = "paid"
		tx.Put("order:"+order.ID, order)
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	env.Queue.Publish(shared.OrderEventsTopic, shared.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_paid"}) // R14
	return shared.OK(200, order)
}
