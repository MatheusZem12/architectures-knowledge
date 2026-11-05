package dip

import "testing"

type spy struct{ to, text string }

func (s *spy) Send(to, text string) { s.to, s.text = to, text }

func TestPayNotifiesWithoutAnyRealMailer(t *testing.T) {
	notifier := &spy{}
	New(notifier).Pay("ana@example.com")
	if notifier.to != "ana@example.com" || notifier.text != "payment confirmed" {
		t.Fatalf("%+v", notifier)
	}
}

// The "real" mailer fits without any change: it already has the Send method.
func TestRealMailerStillFits(t *testing.T) {
	mailer := &SMTPMailerBefore{}
	New(mailer).Pay("ana@example.com")
	if len(mailer.Sent) != 1 {
		t.Fatal(mailer.Sent)
	}
}
