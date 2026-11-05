package srp

import "testing"

func TestBeforeAndAfterGiveTheSameReceipt(t *testing.T) {
	before := (&OrderManagerBefore{}).Process(1250, 3)
	after := ReceiptFormatter{}.Format(Pricing{}.Total(1250, 3))
	if before != after || after != "Total: $37.50" {
		t.Fatalf("before = %q, after = %q", before, after)
	}
}

// The concrete gain: the format can now be tested without going through the calculation or saving anything.
func TestFormatterAlone(t *testing.T) {
	if got := (ReceiptFormatter{}).Format(5); got != "Total: $0.05" {
		t.Fatal(got)
	}
}
