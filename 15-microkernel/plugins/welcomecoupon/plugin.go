// Package welcomecoupon adds the first-order coupon.
package welcomecoupon

import "microkernel/kernel"

type Plugin struct{}

func (Plugin) Name() string { return "welcome-coupon" }

func (p Plugin) Install(registry *kernel.Registry) { registry.AddCoupon("WELCOME10", p) }

// Discount applies the first half of R08.
func (Plugin) Discount(subtotalCents int, isFirstOrder bool) (int, error) {
	if !isFirstOrder {
		return 0, kernel.Fail("coupon_not_eligible", "coupon valid only on the first order")
	}
	return subtotalCents * 10 / 100, nil
}
