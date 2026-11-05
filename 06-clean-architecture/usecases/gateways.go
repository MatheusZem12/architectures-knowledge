// Package usecases is the second ring: one use case per file, each declaring only what
// it needs. The folder name tells what the system does, not how it is built.
package usecases

import "cleanarch/entities"

// The interfaces are small on purpose (Interface Segregation): CancelOrder does not
// receive a "repository of everything", only the three capabilities it actually uses.

type ProductWriter interface {
	AddProduct(entities.Product) entities.Product
}

type ProductReader interface {
	FindProduct(id string) (entities.Product, bool)
}

type ProductSaver interface {
	ProductReader
	SaveProduct(entities.Product)
}

type CustomerWriter interface {
	AddCustomer(entities.Customer) entities.Customer
}

type CustomerReader interface {
	FindCustomer(id string) (entities.Customer, bool)
}

type EmailChecker interface {
	EmailExists(email string) bool
}

type OrderWriter interface {
	AddOrder(entities.Order) entities.Order
}

type OrderReader interface {
	FindOrder(id string) (entities.Order, bool)
}

type OrderSaver interface {
	OrderReader
	SaveOrder(entities.Order)
}

type OrderHistory interface {
	HasActiveOrder(customerID string) bool
}

type NotificationWriter interface {
	AddNotification(entities.Notification)
}

type NotificationReader interface {
	NotificationsOf(customerID string) []entities.Notification
}

type PaymentGateway interface {
	Charge(amountCents int) bool
}

// Transactor runs fn atomically. In a real database, this would be BEGIN and COMMIT.
// The use case knows it needs atomicity, but not how it is achieved.
type Transactor interface {
	Atomic(fn func() error) error
}
