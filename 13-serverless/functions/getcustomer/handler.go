// Package getcustomer is one deployable function.
package getcustomer

import (
	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var customer shared.Customer
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		if !tx.Get("customer:"+req.PathID, &customer) {
			return shared.Refuse(404, "not_found", "customer not found")
		}
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(200, customer)
}
