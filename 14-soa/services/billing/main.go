// Billing service: a coarse service, grouping charging customers and telling them about it.
package main

import (
	"encoding/json"

	"soa/canonical"
)

const maxChargeCents = 1_000_000

var db = canonical.NewDB()

func main() {
	canonical.Serve("billing", map[string]canonical.Handler{
		"ChargePayment":     chargePayment,
		"SendNotification":  sendNotification,
		"ListNotifications": listNotifications,
	})
}

func chargePayment(payload json.RawMessage) (any, error) {
	var in canonical.Charge
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	// R12
	if in.AmountCents > maxChargeCents {
		return nil, canonical.Fail(402, "payment_declined", "payment declined")
	}
	return map[string]bool{"approved": true}, nil
}

// R14
func sendNotification(payload json.RawMessage) (any, error) {
	var in canonical.Notification
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	id, err := db.NextID("ntf")
	if err != nil {
		return nil, err
	}
	return map[string]bool{"sent": true}, db.Put("notifications", id, in)
}

func listNotifications(payload json.RawMessage) (any, error) {
	var in canonical.ByID
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	list := []canonical.Notification{}
	return list, db.List("notifications", "customer_id", in.ID, &list)
}
