package srp

import "fmt"

// OrderManagerBefore has three reasons to change: the pricing rule, the receipt format and
// how things are stored. A receipt tweak forces you to touch the type that calculates money.
type OrderManagerBefore struct {
	saved []string
}

func (m *OrderManagerBefore) Process(priceCents, quantity int) string {
	total := priceCents * quantity
	receipt := fmt.Sprintf("Total: $%d.%02d", total/100, total%100)
	m.saved = append(m.saved, receipt)
	return receipt
}
