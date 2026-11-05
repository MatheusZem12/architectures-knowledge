// Package integration talks to external systems. It sits at the same level as persistence:
// it is "what lies beneath" the services.
package integration

const maxChargeCents = 1_000_000

type PaymentGateway struct{}

func NewPaymentGateway() *PaymentGateway { return &PaymentGateway{} }

// Charge simulates a real gateway, which declines high amounts.
func (g *PaymentGateway) Charge(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}
