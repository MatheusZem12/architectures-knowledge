// Package sendnotification is not called over HTTP. The queue triggers it (R14).
package sendnotification

import (
	"encoding/json"
	"fmt"

	"serverless/cloud"
	"serverless/functions/shared"
)

func Handler(env cloud.Env, req cloud.Request) cloud.Response {
	var notification shared.Notification
	if err := json.Unmarshal(req.Body, &notification); err != nil {
		return cloud.Response{Status: 500}
	}
	err := env.DB.Transact(func(tx *cloud.Tx) error {
		// One key per notification: a single key holding the whole list would be rewritten on every event
		tx.Put(fmt.Sprintf("notifications:%s:%012d", notification.CustomerID, tx.Sequence()), notification)
		return nil
	})
	if err != nil {
		return cloud.Response{Status: 500}
	}
	return cloud.Response{Status: 204}
}
