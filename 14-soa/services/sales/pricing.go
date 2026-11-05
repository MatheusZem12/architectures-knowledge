package main

import "soa/canonical"

const (
	bigOrderMinimum  = 50_000
	freeShippingFrom = 20_000
	shippingPrice    = 2_000
)

// discountFor applies R08.
func discountFor(coupon string, subtotal int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, canonical.Fail(422, "coupon_not_eligible", "coupon valid only on the first order")
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimum {
			return 0, canonical.Fail(422, "coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, canonical.Fail(422, "invalid_coupon", "invalid coupon")
	}
}

// shippingFor applies R09.
func shippingFor(afterDiscount int) int {
	if afterDiscount >= freeShippingFrom {
		return 0
	}
	return shippingPrice
}
