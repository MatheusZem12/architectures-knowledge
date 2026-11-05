// Package fakepayment simulates a payment provider. A real one would be another plugin of the same shape.
package fakepayment

import "microkernel/kernel"

const maxChargeCents = 1_000_000

type Plugin struct{}

func (Plugin) Name() string { return "fake-payment" }

func (Plugin) Install(registry *kernel.Registry) {
	registry.SetPayment(func(amountCents int) bool {
		return amountCents <= maxChargeCents // R12
	})
}
