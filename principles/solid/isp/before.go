package isp

type Order struct {
	ID         string
	TotalCents int
}

// StoreBefore is the fat interface: everything anyone could ever want.
type StoreBefore interface {
	Add(Order)
	Find(id string) (Order, bool)
	All() []Order
	Delete(id string)
	Update(Order)
	Backup() error
}

// RevenueBefore only needs All, but demands all six methods. Whoever tests it must
// write a fake with all of them, and it breaks if Backup changes, though it never uses it.
func RevenueBefore(store StoreBefore) int {
	total := 0
	for _, order := range store.All() {
		total += order.TotalCents
	}
	return total
}
