// Package payments simulates the payment gateway.
package payments

const maxChargeCents = 1_000_000

// Charge is a plain function: orders calls it directly, with no way to swap it.
func Charge(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}
