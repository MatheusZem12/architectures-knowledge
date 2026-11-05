package entities

type Coupon interface {
	Discount(subtotalCents int, isFirstOrder bool) (int, error)
}

var coupons = map[string]Coupon{
	"WELCOME10": welcomeCoupon{},
	"BIG20":     bigOrderCoupon{},
}

// R08
func discountFor(code string, subtotalCents int, isFirstOrder bool) (int, error) {
	if code == "" {
		return 0, nil
	}
	coupon, ok := coupons[code]
	if !ok {
		return 0, Fail(CodeInvalidCoupon, "invalid coupon")
	}
	return coupon.Discount(subtotalCents, isFirstOrder)
}

type welcomeCoupon struct{}

func (welcomeCoupon) Discount(subtotalCents int, isFirstOrder bool) (int, error) {
	if !isFirstOrder {
		return 0, Fail(CodeCouponNotEligible, "coupon valid only on the first order")
	}
	return subtotalCents * 10 / 100, nil
}

type bigOrderCoupon struct{}

func (bigOrderCoupon) Discount(subtotalCents int, _ bool) (int, error) {
	if subtotalCents < 50_000 {
		return 0, Fail(CodeCouponNotEligible, "subtotal below the coupon minimum")
	}
	return subtotalCents * 20 / 100, nil
}
