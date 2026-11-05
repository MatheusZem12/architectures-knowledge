package dip

import "fmt"

// SMTPMailerBefore is a low-level detail. Imagine it opens a real connection.
type SMTPMailerBefore struct{ Sent []string }

func (m *SMTPMailerBefore) Send(to, text string) {
	m.Sent = append(m.Sent, fmt.Sprintf("%s: %s", to, text))
}

// OrdersBefore is the business rule (high level), and it creates the detail by itself. It
// depends on SMTP to compile and to run: you cannot test it without sending email, nor
// switch to SMS without editing this type.
type OrdersBefore struct {
	mailer *SMTPMailerBefore
}

func NewOrdersBefore() *OrdersBefore { return &OrdersBefore{mailer: &SMTPMailerBefore{}} }

func (o *OrdersBefore) Pay(customer string) { o.mailer.Send(customer, "payment confirmed") }
