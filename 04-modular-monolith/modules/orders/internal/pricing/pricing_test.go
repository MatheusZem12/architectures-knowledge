package pricing

import (
	"errors"
	"testing"
)

// No HTTP and no store: the rule tests on its own, since it depends on nothing.
func TestDiscount(t *testing.T) {
	cases := []struct {
		coupon   string
		subtotal int
		first    bool
		want     int
		err      error
	}{
		{"", 1000, true, 0, nil},
		{"WELCOME10", 999, true, 99, nil},
		{"WELCOME10", 999, false, 0, ErrNotEligible},
		{"BIG20", 50_000, false, 10_000, nil},
		{"BIG20", 49_999, false, 0, ErrNotEligible},
		{"XPTO", 1000, true, 0, ErrInvalidCoupon},
	}
	for _, c := range cases {
		got, err := Discount(c.coupon, c.subtotal, c.first)
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("Discount(%q, %d, %v) = %d, %v", c.coupon, c.subtotal, c.first, got, err)
		}
	}
}
