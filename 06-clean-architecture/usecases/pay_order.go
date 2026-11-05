package usecases

import "cleanarch/entities"

type PayOrder struct {
	orders        OrderSaver
	notifications NotificationWriter
	payments      PaymentGateway
	tx            Transactor
}

func NewPayOrder(orders OrderSaver, notifications NotificationWriter, payments PaymentGateway, tx Transactor) *PayOrder {
	return &PayOrder{orders: orders, notifications: notifications, payments: payments, tx: tx}
}

func (uc *PayOrder) Execute(id string) (OrderOutput, error) {
	var out OrderOutput
	err := uc.tx.Atomic(func() error {
		order, ok := uc.orders.FindOrder(id)
		if !ok {
			return entities.Fail(entities.CodeNotFound, "order not found")
		}
		if err := order.CanPay(); err != nil {
			return err
		}
		// R12
		if !uc.payments.Charge(order.TotalCents) {
			return entities.Fail(entities.CodePaymentDeclined, "payment declined")
		}
		order.MarkPaid()
		uc.orders.SaveOrder(order)
		uc.notifications.AddNotification(entities.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: "order_paid"}) // R14
		out = orderOutput(order)
		return nil
	})
	return out, err
}
