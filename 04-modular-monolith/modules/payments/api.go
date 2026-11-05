// Package payments is the payments module. Today it simulates an external gateway.
package payments

import "modular/platform/web"

const maxChargeCents = 1_000_000

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Charge(amountCents int) error {
	// R12
	if amountCents > maxChargeCents {
		return web.Fail("payment_declined", "payment declined")
	}
	return nil
}
