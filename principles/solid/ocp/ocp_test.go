package ocp

import "testing"

// fixedCoupon lives in the test on purpose: a coupon of a brand new type works without
// any line of after.go having changed.
type fixedCoupon struct{ cents int }

func (c fixedCoupon) Discount(int) int { return c.cents }

func TestExtensionWithoutModification(t *testing.T) {
	calculator := NewCalculator(map[string]Coupon{
		"WELCOME10": PercentCoupon{10},
		"BIG20":     PercentCoupon{20},
		"MENOS5":    fixedCoupon{500},
	})
	for _, code := range []string{"WELCOME10", "BIG20", "XPTO"} {
		if got, want := calculator.Discount(code, 1000), DiscountBefore(code, 1000); got != want {
			t.Errorf("%s: after = %d, before = %d", code, got, want)
		}
	}
	if got := calculator.Discount("MENOS5", 1000); got != 500 {
		t.Fatalf("MENOS5 = %d", got)
	}
}
