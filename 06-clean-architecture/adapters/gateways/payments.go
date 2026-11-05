package gateways

const maxChargeCents = 1_000_000

type FakePayments struct{}

func NewFakePayments() FakePayments { return FakePayments{} }

func (FakePayments) Charge(amountCents int) bool {
	return amountCents <= maxChargeCents // R12
}
