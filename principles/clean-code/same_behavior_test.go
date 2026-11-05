package cleancode

import "testing"

// The refactoring is legitimate because this test proves nothing changed from the outside: both
// versions must give the same result for every combination.
func TestBeforeAndAfterBehaveTheSame(t *testing.T) {
	prices := []int{1, 999, 19_999, 20_000, 21_000, 49_999, 50_000, 1_000_000}
	coupons := []string{"", welcomeCoupon, bigOrderCoupon, "NOSUCHCOUPON"}
	for _, price := range prices {
		for quantity := 1; quantity <= 3; quantity++ {
			for _, coupon := range coupons {
				for _, first := range []bool{true, false} {
					s, d, sh, total := calc([]map[string]int{{"p": price, "q": quantity}}, coupon, first)
					got := CalculateTotals([]Item{{price, quantity}}, coupon, first)
					if want := (Totals{s, d, sh, total}); got != want {
						t.Errorf("price=%d qty=%d coupon=%q first=%v: %+v != %+v", price, quantity, coupon, first, got, want)
					}
				}
			}
		}
	}
}
