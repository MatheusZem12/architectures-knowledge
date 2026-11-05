// Package bigordercoupon adds the coupon for large orders.
package bigordercoupon

import "microkernel/kernel"

const minimumCents = 50_000

type Plugin struct{}

func (Plugin) Name() string { return "big-order-coupon" }

func (p Plugin) Install(registry *kernel.Registry) { registry.AddCoupon("BIG20", p) }

// Discount applies the second half of R08.
func (Plugin) Discount(subtotalCents int, _ bool) (int, error) {
	if subtotalCents < minimumCents {
		return 0, kernel.Fail("coupon_not_eligible", "subtotal below the coupon minimum")
	}
	return subtotalCents * 20 / 100, nil
}
