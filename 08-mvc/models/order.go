package models

const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusCancelled = "cancelled"

	maxQuantityPerItem    = 10
	bigOrderMinimumCents  = 50_000
	freeShippingFromCents = 20_000
	shippingCents         = 2_000
)

type Line struct {
	ProductID string
	Quantity  int
}

type OrderItem struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

type Order struct {
	ID            string
	CustomerID    string
	Status        string
	Items         []OrderItem
	Coupon        string
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

// Discount is pure, so it can be tested with nothing around it (R08).
func Discount(coupon string, subtotalCents int, isFirstOrder bool) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		if !isFirstOrder {
			return 0, Fail("coupon_not_eligible", "coupon valid only on the first order")
		}
		return subtotalCents * 10 / 100, nil
	case "BIG20":
		if subtotalCents < bigOrderMinimumCents {
			return 0, Fail("coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotalCents * 20 / 100, nil
	default:
		return 0, Fail("invalid_coupon", "invalid coupon")
	}
}

func (db *DB) PlaceOrder(customerID string, lines []Line, coupon string) (Order, error) {
	// R04
	if len(lines) == 0 {
		return Order{}, Fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Quantity < 1 || line.Quantity > maxQuantityPerItem || seen[line.ProductID] {
			return Order{}, Fail("validation", "item with invalid quantity or repeated")
		}
		seen[line.ProductID] = true
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	// R05
	if _, ok := db.customers[customerID]; !ok {
		return Order{}, Fail("customer_not_found", "customer does not exist")
	}

	// R05, R06 and R07: check everything before changing any stock
	order := Order{CustomerID: customerID, Status: StatusPending, Coupon: coupon}
	for _, line := range lines {
		product, ok := db.products[line.ProductID]
		if !ok {
			return Order{}, Fail("product_not_found", "product does not exist")
		}
		if product.Stock < line.Quantity {
			return Order{}, Fail("insufficient_stock", "insufficient stock")
		}
		order.Items = append(order.Items, OrderItem{product.ID, line.Quantity, product.PriceCents})
		order.SubtotalCents += product.PriceCents * line.Quantity
	}

	discount, err := Discount(coupon, order.SubtotalCents, !db.hasActiveOrder(customerID))
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

	// R06: only save the stock once nothing else can fail
	for _, item := range order.Items {
		product := db.products[item.ProductID]
		product.Stock -= item.Quantity
		db.products[item.ProductID] = product
	}
	order.ID = db.nextID("ord")
	db.orders[order.ID] = order
	db.notify(order, "order_created")
	return order, nil
}

func (db *DB) hasActiveOrder(customerID string) bool {
	for _, order := range db.orders {
		if order.CustomerID == customerID && order.Status != StatusCancelled {
			return true
		}
	}
	return false
}

func (db *DB) FindOrder(id string) (Order, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.findOrder(id)
}

func (db *DB) findOrder(id string) (Order, error) {
	order, ok := db.orders[id]
	if !ok {
		return Order{}, Fail("not_found", "order not found")
	}
	return order, nil
}

func (db *DB) PayOrder(id string) (Order, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	order, err := db.findOrder(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status != StatusPending {
		return Order{}, Fail("invalid_transition", "only a pending order can be paid")
	}
	if !db.gateway(order.TotalCents) {
		return Order{}, Fail("payment_declined", "payment declined")
	}
	order.Status = StatusPaid
	db.orders[id] = order
	db.notify(order, "order_paid")
	return order, nil
}

func (db *DB) CancelOrder(id string) (Order, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	order, err := db.findOrder(id)
	if err != nil {
		return Order{}, err
	}
	// R11
	if order.Status == StatusCancelled {
		return Order{}, Fail("invalid_transition", "order already cancelled")
	}
	// R13
	for _, item := range order.Items {
		product := db.products[item.ProductID]
		product.Stock += item.Quantity
		db.products[item.ProductID] = product
	}
	order.Status = StatusCancelled
	db.orders[id] = order
	db.notify(order, "order_cancelled")
	return order, nil
}
