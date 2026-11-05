package srp

import "fmt"

// Each type now has a single reason to change, meaning it answers to a single "person".

// Pricing changes when the pricing rule changes (sales asks).
type Pricing struct{}

func (Pricing) Total(priceCents, quantity int) int { return priceCents * quantity }

// ReceiptFormatter changes when the receipt text changes (design asks).
type ReceiptFormatter struct{}

func (ReceiptFormatter) Format(totalCents int) string {
	return fmt.Sprintf("Total: $%d.%02d", totalCents/100, totalCents%100)
}

// ReceiptLog changes when the way of storing changes (infrastructure asks).
type ReceiptLog struct {
	saved []string
}

func (l *ReceiptLog) Save(receipt string) { l.saved = append(l.saved, receipt) }
