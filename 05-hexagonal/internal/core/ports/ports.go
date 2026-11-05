// Package ports defines the hexagon's sockets: the contracts between the core and the world.
//
// The interfaces live here, next to the core, not with whoever implements them. That is the
// inversion that makes the database depend on the application, instead of the other way around.
package ports

import "hexagonal/internal/core/domain"

// ---------- inbound (driving) port: how the world calls the core ----------

type Shop interface {
	CreateProduct(name string, priceCents, stock int) (domain.Product, error)
	GetProduct(id string) (domain.Product, error)
	CreateCustomer(name, email string) (domain.Customer, error)
	GetCustomer(id string) (domain.Customer, error)
	Notifications(customerID string) []domain.Notification
	PlaceOrder(customerID string, lines []domain.Line, coupon string) (domain.Order, error)
	GetOrder(id string) (domain.Order, error)
	PayOrder(id string) (domain.Order, error)
	CancelOrder(id string) (domain.Order, error)
}

// ---------- outbound (driven) ports: how the core calls the world ----------

type ProductRepository interface {
	AddProduct(p domain.Product) domain.Product
	FindProduct(id string) (domain.Product, bool)
	SaveProduct(p domain.Product)
}

type CustomerRepository interface {
	AddCustomer(c domain.Customer) domain.Customer
	FindCustomer(id string) (domain.Customer, bool)
	EmailExists(email string) bool
}

type OrderRepository interface {
	AddOrder(o domain.Order) domain.Order
	FindOrder(id string) (domain.Order, bool)
	SaveOrder(o domain.Order)
	HasActiveOrder(customerID string) bool
}

type NotificationRepository interface {
	AddNotification(n domain.Notification)
	NotificationsOf(customerID string) []domain.Notification
}

type PaymentGateway interface {
	Charge(amountCents int) bool
}
