// Package listnotifications is one deployable function.
package listnotifications

import (
	"encoding/json"

	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	list := []shared.Notification{}
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		for _, raw := range tx.Scan("notifications:" + req.PathID + ":") {
			var notification shared.Notification
			if err := json.Unmarshal(raw, &notification); err != nil {
				return err
			}
			list = append(list, notification)
		}
		return nil
	})
	if err != nil {
		return shared.Fail(err)
	}
	return shared.OK(200, list)
}
