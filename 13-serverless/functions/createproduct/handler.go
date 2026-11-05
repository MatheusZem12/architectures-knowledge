// Package createproduct is one deployable function.
package createproduct

import (
	"strings"

	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var in shared.Product
	if response, ok := shared.Decode(req, &in); !ok {
		return response
	}
	// R01
	if strings.TrimSpace(in.Name) == "" || in.PriceCents <= 0 || in.Stock < 0 {
		return shared.Fail(shared.Refuse(422, "validation", "product needs a name, a positive price and non-negative stock"))
	}
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		in.ID = tx.NextID("prod")
		tx.Put("product:"+in.ID, in)
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(201, in)
}
