package usecases

import "cleanarch/entities"

type PlaceOrderItem struct {
	ProductID string
	Quantity  int
}

type PlaceOrderInput struct {
	CustomerID string
	Items      []PlaceOrderItem
	Coupon     string
}

type PlaceOrder struct {
	customers     CustomerReader
	products      ProductSaver
	orders        OrderWriter
	history       OrderHistory
	notifications NotificationWriter
	tx            Transactor
}

func NewPlaceOrder(
	customers CustomerReader,
	products ProductSaver,
	orders OrderWriter,
	history OrderHistory,
	notifications NotificationWriter,
	tx Transactor,
) *PlaceOrder {
	return &PlaceOrder{customers, products, orders, history, notifications, tx}
}

func (uc *PlaceOrder) Execute(in PlaceOrderInput) (OrderOutput, error) {
	lines := make([]entities.Line, len(in.Items))
	for i, item := range in.Items {
		lines[i] = entities.Line{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	if err := entities.ValidateLines(lines); err != nil {
		return OrderOutput{}, err
	}

	var out OrderOutput
	err := uc.tx.Atomic(func() error {
		// R05
		if _, ok := uc.customers.FindCustomer(in.CustomerID); !ok {
			return entities.Fail(entities.CodeCustomerNotFound, "customer does not exist")
		}
		// R06: reservations happen on copies. Nothing is saved until the end.
		items := make([]entities.OrderItem, 0, len(lines))
		reserved := make([]entities.Product, 0, len(lines))
		for _, line := range lines {
			product, ok := uc.products.FindProduct(line.ProductID)
			if !ok {
				return entities.Fail(entities.CodeProductNotFound, "product does not exist")
			}
			if err := product.Reserve(line.Quantity); err != nil {
				return err
			}
			reserved = append(reserved, product)
			items = append(items, entities.OrderItem{ProductID: product.ID, Quantity: line.Quantity, UnitPriceCents: product.PriceCents}) // R07
		}

		order, err := entities.NewOrder(in.CustomerID, items, in.Coupon, !uc.history.HasActiveOrder(in.CustomerID))
		if err != nil {
			return err
		}
		for _, product := range reserved {
			uc.products.SaveProduct(product)
		}
		order = uc.orders.AddOrder(order)
		uc.notifications.AddNotification(entities.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_created"}) // R14
		out = orderOutput(order)
		return nil
	})
	return out, err
}
