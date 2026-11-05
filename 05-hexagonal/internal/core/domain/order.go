package domain

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusCancelled Status = "cancelled"
)

const (
	maxQuantityPerItem    = 10
	freeShippingFromCents = 20_000
	shippingCents         = 2_000
)

type OrderItem struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

// Order is a rich model: compare with folder 02, where an order is just a bag of fields.
// Whoever wants to change the status must go through Pay or Cancel, and the rules come along.
type Order struct {
	ID            string
	CustomerID    string
	Status        Status
	Items         []OrderItem
	Coupon        string
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

// Line is what the customer asked for, before it becomes an item with a price.
type Line struct {
	ProductID string
	Quantity  int
}

// ValidateLines applies R04.
func ValidateLines(lines []Line) error {
	if len(lines) == 0 {
		return Fail(CodeValidation, "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Quantity < 1 || line.Quantity > maxQuantityPerItem || seen[line.ProductID] {
			return Fail(CodeValidation, "item with invalid quantity or repeated")
		}
		seen[line.ProductID] = true
	}
	return nil
}

// NewOrder calculates all the order's amounts. items already carries the price at purchase time (R07).
func NewOrder(customerID string, items []OrderItem, coupon string, isFirstOrder bool) (Order, error) {
	order := Order{CustomerID: customerID, Status: StatusPending, Items: items, Coupon: coupon}
	for _, item := range items {
		order.SubtotalCents += item.UnitPriceCents * item.Quantity
	}
	discount, err := discountFor(coupon, order.SubtotalCents, isFirstOrder)
	if err != nil {
		return Order{}, err
	}
	order.DiscountCents = discount
	// R09
	order.ShippingCents = shippingCents
	if order.SubtotalCents-order.DiscountCents >= freeShippingFromCents {
		order.ShippingCents = 0
	}
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents
	return order, nil
}

// CanPay and MarkPaid are separate because the gateway is called between them.
func (o *Order) CanPay() error {
	// R11
	if o.Status != StatusPending {
		return Fail(CodeInvalidTransition, "only a pending order can be paid")
	}
	return nil
}

func (o *Order) MarkPaid() { o.Status = StatusPaid }

func (o *Order) Cancel() error {
	// R11
	if o.Status == StatusCancelled {
		return Fail(CodeInvalidTransition, "order already cancelled")
	}
	o.Status = StatusCancelled
	return nil
}
