package ocp

// Coupon is the extension point: a new coupon is a new type, without touching existing code.
type Coupon interface {
	Discount(subtotal int) int
}

type PercentCoupon struct{ Percent int }

func (c PercentCoupon) Discount(subtotal int) int { return subtotal * c.Percent / 100 }

type Calculator struct {
	coupons map[string]Coupon
}

func NewCalculator(coupons map[string]Coupon) *Calculator {
	return &Calculator{coupons: coupons}
}

// Discount is closed for modification: it never needs to change again.
func (c *Calculator) Discount(code string, subtotal int) int {
	coupon, ok := c.coupons[code]
	if !ok {
		return 0
	}
	return coupon.Discount(subtotal)
}
