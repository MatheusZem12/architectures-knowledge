package isp

import "testing"

// For the "before", the fake needs six methods, five of them useless to the test.
type fatFake struct{ orders []Order }

func (f fatFake) Add(Order)                 {}
func (f fatFake) Find(string) (Order, bool) { return Order{}, false }
func (f fatFake) All() []Order              { return f.orders }
func (f fatFake) Delete(string)             {}
func (f fatFake) Update(Order)              {}
func (f fatFake) Backup() error             { return nil }

// For the "after", one line does it.
type slimFake []Order

func (f slimFake) All() []Order { return f }

func TestBothCalculateTheSame(t *testing.T) {
	orders := []Order{{"a", 100}, {"b", 250}}
	if before, after := RevenueBefore(fatFake{orders}), Revenue(slimFake(orders)); before != after || after != 350 {
		t.Fatalf("before = %d, after = %d", before, after)
	}
}

// The fat type still works wherever the small interface is requested.
func TestFatTypeSatisfiesSlimInterface(t *testing.T) {
	var _ OrderLister = fatFake{}
}
