package usecases

import "cleanarch/entities"

type CancelOrder struct {
	orders        OrderSaver
	products      ProductSaver
	notifications NotificationWriter
	tx            Transactor
}

func NewCancelOrder(orders OrderSaver, products ProductSaver, notifications NotificationWriter, tx Transactor) *CancelOrder {
	return &CancelOrder{orders: orders, products: products, notifications: notifications, tx: tx}
}

func (uc *CancelOrder) Execute(id string) (OrderOutput, error) {
	var out OrderOutput
	err := uc.tx.Atomic(func() error {
		order, ok := uc.orders.FindOrder(id)
		if !ok {
			return entities.Fail(entities.CodeNotFound, "order not found")
		}
		if err := order.Cancel(); err != nil {
			return err
		}
		// R13
		for _, item := range order.Items {
			if product, ok := uc.products.FindProduct(item.ProductID); ok {
				product.Release(item.Quantity)
				uc.products.SaveProduct(product)
			}
		}
		uc.orders.SaveOrder(order)
		uc.notifications.AddNotification(entities.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_cancelled"}) // R14
		out = orderOutput(order)
		return nil
	})
	return out, err
}
