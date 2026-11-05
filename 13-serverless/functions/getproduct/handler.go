// Package getproduct is one deployable function.
package getproduct

import (
	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var product shared.Product
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		if !tx.Get("product:"+req.PathID, &product) {
			return shared.Refuse(404, "not_found", "product not found")
		}
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(200, product)
}
