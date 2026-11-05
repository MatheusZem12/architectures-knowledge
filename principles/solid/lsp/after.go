package lsp

// The interface now only promises what everyone delivers. Refunding became a separate capability.
type Charger interface {
	Charge(cents int)
}

type Refunder interface {
	Refund(cents int)
}

type Card struct{ Balance int }

func (c *Card) Charge(cents int) { c.Balance += cents }
func (c *Card) Refund(cents int) { c.Balance -= cents }

type GiftCard struct{ Balance int }

func (g *GiftCard) Charge(cents int) { g.Balance += cents }

// Cancel only accepts what can refund. Passing a GiftCard is now a compile error:
// the problem moved from production to the compiler.
func Cancel(gateway Refunder, cents int) { gateway.Refund(cents) }

var (
	_ Charger  = (*Card)(nil)
	_ Refunder = (*Card)(nil)
	_ Charger  = (*GiftCard)(nil)
)
