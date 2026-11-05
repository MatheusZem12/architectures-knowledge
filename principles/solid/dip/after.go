package dip

// Notifier is the abstraction, and it belongs to the business rule: the rule says what it needs.
// SMTP now depends on this interface, not the other way around. Hence "inversion".
type Notifier interface {
	Send(to, text string)
}

type Orders struct {
	notifier Notifier
}

// New receives the detail from outside (dependency injection).
func New(notifier Notifier) *Orders { return &Orders{notifier: notifier} }

func (o *Orders) Pay(customer string) { o.notifier.Send(customer, "payment confirmed") }
