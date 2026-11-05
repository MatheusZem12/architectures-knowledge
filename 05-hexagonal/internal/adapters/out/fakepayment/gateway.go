// Package fakepayment is an outbound adapter: a pretend gateway. An adapter for a real
// provider would have the same shape, implementing ports.PaymentGateway.
package fakepayment

const maxChargeCents = 1_000_000

type Gateway struct{}

func New() Gateway { return Gateway{} }

func (Gateway) Charge(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}
