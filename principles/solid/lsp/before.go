package lsp

// GatewayBefore promises to charge and refund.
type GatewayBefore interface {
	Charge(cents int)
	Refund(cents int)
}

type CardBefore struct{ Balance int }

func (c *CardBefore) Charge(cents int) { c.Balance += cents }
func (c *CardBefore) Refund(cents int) { c.Balance -= cents }

// GiftCardBefore compiles, but breaks the promise: gift cards have no refund. It cannot
// substitute the type it claims to implement, and that only shows up at runtime.
type GiftCardBefore struct{ Balance int }

func (g *GiftCardBefore) Charge(cents int) { g.Balance += cents }
func (g *GiftCardBefore) Refund(int)       { panic("gift cards have no refund") }

// CancelBefore trusts the interface, rightly so, and blows up on the gift card.
func CancelBefore(gateway GatewayBefore, cents int) { gateway.Refund(cents) }
