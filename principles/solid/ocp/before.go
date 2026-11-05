package ocp

// DiscountBefore must be opened and edited for every new coupon. The file grows longer
// and each edit risks breaking the coupons that already worked.
func DiscountBefore(coupon string, subtotal int) int {
	switch coupon {
	case "WELCOME10":
		return subtotal * 10 / 100
	case "BIG20":
		return subtotal * 20 / 100
	default:
		return 0
	}
}
