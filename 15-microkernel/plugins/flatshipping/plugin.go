// Package flatshipping charges a flat fee, with free shipping above a threshold.
package flatshipping

import "microkernel/kernel"

const (
	freeFromCents = 20_000
	feeCents      = 2_000
)

type Plugin struct{}

func (Plugin) Name() string { return "flat-shipping" }

func (Plugin) Install(registry *kernel.Registry) {
	registry.SetShipping(func(subtotalAfterDiscountCents int) int {
		// R09
		if subtotalAfterDiscountCents >= freeFromCents {
			return 0
		}
		return feeCents
	})
}
