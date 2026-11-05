package cleancode

const (
	welcomeCoupon  = "WELCOME10"
	bigOrderCoupon = "BIG20"

	bigOrderMinimumCents  = 50_000
	freeShippingFromCents = 20_000
	flatShippingCents     = 2_000
)

type Item struct {
	UnitPriceCents int
	Quantity       int
}

type Totals struct {
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

// CalculateTotals reads like the business rule: subtotal, discount, shipping, total.
func CalculateTotals(items []Item, coupon string, isFirstOrder bool) Totals {
	subtotal := subtotalOf(items)
	discount := discountFor(coupon, subtotal, isFirstOrder)
	shipping := shippingFor(subtotal - discount)
	return Totals{subtotal, discount, shipping, subtotal - discount + shipping}
}

func subtotalOf(items []Item) int {
	subtotal := 0
	for _, item := range items {
		subtotal += item.UnitPriceCents * item.Quantity
	}
	return subtotal
}

func discountFor(coupon string, subtotal int, isFirstOrder bool) int {
	switch {
	case coupon == welcomeCoupon && isFirstOrder:
		return percentOf(subtotal, 10)
	case coupon == bigOrderCoupon && subtotal >= bigOrderMinimumCents:
		return percentOf(subtotal, 20)
	default:
		return 0
	}
}

func shippingFor(valueAfterDiscount int) int {
	if valueAfterDiscount >= freeShippingFromCents {
		return 0
	}
	return flatShippingCents
}

// percentOf rounds down, because the store never gives a discount in fractions of a cent.
func percentOf(value, percent int) int {
	return value * percent / 100
}
