package lsp

import "testing"

func TestBeforeBreaksAtRuntime(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("the gift card should have blown up")
		}
	}()
	CancelBefore(&GiftCardBefore{}, 100)
}

func TestAfterWorksForWhoCanRefund(t *testing.T) {
	card := &Card{Balance: 100}
	Cancel(card, 100)
	if card.Balance != 0 {
		t.Fatalf("balance = %d", card.Balance)
	}
}
