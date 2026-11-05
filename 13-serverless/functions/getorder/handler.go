// Package getorder is one deployable function.
package getorder

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
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(200, order)
}
