package isp

// OrderLister is only what Revenue uses. In Go, the interface lives with the consumer, and the
// concrete type satisfies it without declaring anything.
type OrderLister interface {
	All() []Order
}

func Revenue(orders OrderLister) int {
	total := 0
	for _, order := range orders.All() {
		total += order.TotalCents
	}
	return total
}
