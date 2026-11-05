// Package pricing calculates an order's amounts. It is pure logic, with no dependencies.
package pricing

import "errors"

var (
	ErrInvalidCoupon = errors.New("invalid coupon")
	ErrNotEligible   = errors.New("customer or order does not meet the coupon conditions")
)

const (
	bigOrderMinimum  = 50_000
	freeShippingFrom = 20_000
	shippingPrice    = 2_000
)

// Discount calculates the discount in cents (R08).
func Discount(coupon string, subtotal int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, ErrNotEligible
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimum {
			return 0, ErrNotEligible
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, ErrInvalidCoupon
	}
}

// Shipping calculates shipping on the amount after discount (R09).
func Shipping(subtotalAfterDiscount int) int {
	if subtotalAfterDiscount >= freeShippingFrom {
		return 0
	}
	return shippingPrice
}
