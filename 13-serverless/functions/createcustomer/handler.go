// Package createcustomer is one deployable function.
package createcustomer

import (
	"strings"

	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var in shared.Customer
	if response, ok := shared.Decode(req, &in); !ok {
		return response
	}
	// R02
	at := strings.Index(in.Email, "@")
	if strings.TrimSpace(in.Name) == "" || at <= 0 || at == len(in.Email)-1 {
		return shared.Fail(shared.Refuse(422, "validation", "customer needs a name and a valid email"))
	}
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		// R03: the email itself is a key, because a key-value database has no unique index
		emailKey := "email:" + strings.ToLower(in.Email)
		var taken bool
		if tx.Get(emailKey, &taken) {
			return shared.Refuse(409, "email_taken", "email already registered")
		}
		in.ID = tx.NextID("cust")
		tx.Put(emailKey, true)
		tx.Put("customer:"+in.ID, in)
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(201, in)
}
