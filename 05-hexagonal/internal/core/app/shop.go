// Package app orchestrates the use cases: reads through ports, calls the domain, writes through ports.
// Almost no rule lives here, only the sequence of steps.
package app

import (
	"sync"

	"hexagonal/internal/core/domain"
	"hexagonal/internal/core/ports"
)

type Shop struct {
	mu            sync.Mutex // in a real database, this would be a transaction
	products      ports.ProductRepository
	customers     ports.CustomerRepository
	orders        ports.OrderRepository
	notifications ports.NotificationRepository
	payments      ports.PaymentGateway
}

func NewShop(
	products ports.ProductRepository,
	customers ports.CustomerRepository,
	orders ports.OrderRepository,
	notifications ports.NotificationRepository,
	payments ports.PaymentGateway,
) *Shop {
	return &Shop{products: products, customers: customers, orders: orders, notifications: notifications, payments: payments}
}

var _ ports.Shop = (*Shop)(nil)

func (s *Shop) CreateProduct(name string, priceCents, stock int) (domain.Product, error) {
	product, err := domain.NewProduct(name, priceCents, stock)
	if err != nil {
		return domain.Product{}, err
	}
	return s.products.AddProduct(product), nil
}

func (s *Shop) GetProduct(id string) (domain.Product, error) {
	product, ok := s.products.FindProduct(id)
	if !ok {
		return domain.Product{}, domain.Fail(domain.CodeNotFound, "product not found")
	}
	return product, nil
}

func (s *Shop) CreateCustomer(name, email string) (domain.Customer, error) {
	customer, err := domain.NewCustomer(name, email)
	if err != nil {
		return domain.Customer{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// R03
	if s.customers.EmailExists(email) {
		return domain.Customer{}, domain.Fail(domain.CodeEmailTaken, "email already registered")
	}
	return s.customers.AddCustomer(customer), nil
}

func (s *Shop) GetCustomer(id string) (domain.Customer, error) {
	customer, ok := s.customers.FindCustomer(id)
	if !ok {
		return domain.Customer{}, domain.Fail(domain.CodeNotFound, "customer not found")
	}
	return customer, nil
}

func (s *Shop) Notifications(customerID string) []domain.Notification {
	return s.notifications.NotificationsOf(customerID)
}

func (s *Shop) PlaceOrder(customerID string, lines []domain.Line, coupon string) (domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := domain.ValidateLines(lines); err != nil {
		return domain.Order{}, err
	}
	// R05
	if _, ok := s.customers.FindCustomer(customerID); !ok {
		return domain.Order{}, domain.Fail(domain.CodeCustomerNotFound, "customer does not exist")
	}

	// R06: reservations happen on copies. Nothing is saved until the end.
	items := make([]domain.OrderItem, 0, len(lines))
	reserved := make([]domain.Product, 0, len(lines))
	for _, line := range lines {
		product, ok := s.products.FindProduct(line.ProductID)
		if !ok {
			return domain.Order{}, domain.Fail(domain.CodeProductNotFound, "product does not exist")
		}
		if err := product.Reserve(line.Quantity); err != nil {
			return domain.Order{}, err
		}
		reserved = append(reserved, product)
		items = append(items, domain.OrderItem{ProductID: product.ID, Quantity: line.Quantity, UnitPriceCents: product.PriceCents}) // R07
	}

	order, err := domain.NewOrder(customerID, items, coupon, !s.orders.HasActiveOrder(customerID))
	if err != nil {
		return domain.Order{}, err
	}

	for _, product := range reserved {
		s.products.SaveProduct(product)
	}
	order = s.orders.AddOrder(order)
	s.notify(order, "order_created") // R14
	return order, nil
}

func (s *Shop) GetOrder(id string) (domain.Order, error) {
	order, ok := s.orders.FindOrder(id)
	if !ok {
		return domain.Order{}, domain.Fail(domain.CodeNotFound, "order not found")
	}
	return order, nil
}

func (s *Shop) PayOrder(id string) (domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, err := s.GetOrder(id)
	if err != nil {
		return domain.Order{}, err
	}
	if err := order.CanPay(); err != nil {
		return domain.Order{}, err
	}
	// R12: the gateway, behind the port, decides whether to decline
	if !s.payments.Charge(order.TotalCents) {
		return domain.Order{}, domain.Fail(domain.CodePaymentDeclined, "payment declined")
	}
	order.MarkPaid()
	s.orders.SaveOrder(order)
	s.notify(order, "order_paid") // R14
	return order, nil
}

func (s *Shop) CancelOrder(id string) (domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, err := s.GetOrder(id)
	if err != nil {
		return domain.Order{}, err
	}
	if err := order.Cancel(); err != nil {
		return domain.Order{}, err
	}
	for _, item := range order.Items {
		if product, ok := s.products.FindProduct(item.ProductID); ok {
			product.Release(item.Quantity)
			s.products.SaveProduct(product)
		}
	}
	s.orders.SaveOrder(order)
	s.notify(order, "order_cancelled") // R14
	return order, nil
}

func (s *Shop) notify(order domain.Order, kind string) {
	s.notifications.AddNotification(domain.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: kind})
}
