// Package cancelorder is one deployable function.
package cancelorder

import (
	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var order shared.Order
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		if !tx.Get("order:"+req.PathID, &order) {
			return shared.Refuse(404, "not_found", "order not found")
		}
		// R11
		if order.Status == "cancelled" {
			return shared.Refuse(409, "invalid_transition", "order already cancelled")
		}
		// R13
		for _, item := range order.Items {
			var product shared.Product
			if tx.Get("product:"+item.ProductID, &product) {
				product.Stock += item.Quantity
				tx.Put("product:"+product.ID, product)
			}
		}
		activeKey := "active-orders:" + order.CustomerID
		var active int
		tx.Get(activeKey, &active)
		tx.Put(activeKey, active-1)
		order.Status = "cancelled"
		tx.Put("order:"+order.ID, order)
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	env.Queue.Publish(shared.OrderEventsTopic, shared.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_cancelled"}) // R14
	return shared.OK(200, order)
}
