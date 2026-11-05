package main

import "testing"

func TestShippingUsesTheValueAfterDiscount(t *testing.T) {
	discount, err := discountFor("WELCOME10", 21_000, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := shippingFor(21_000 - discount); got != shippingPrice {
		t.Fatalf("shipping = %d", got)
	}
}
