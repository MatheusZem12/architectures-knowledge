package service

import (
	"sync"

	"layered/internal/integration"
	"layered/internal/model"
	"layered/internal/repository"
)

const (
	maxQuantityPerItem    = 10
	bigOrderMinimumCents  = 50_000
	freeShippingFromCents = 20_000
	shippingCents         = 2_000
)

type ItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type PlaceOrderInput struct {
	CustomerID string      `json:"customer_id"`
	Items      []ItemInput `json:"items"`
	Coupon     string      `json:"coupon"`
}

// OrderService is the classic symptom of this architecture: a service that knows everything.
// It depends on five concrete pieces. Note that the types are *repository.X and not
// interfaces: the business layer points directly at the persistence layer.
type OrderService struct {
	mu            sync.Mutex // in a real database, this would be a transaction
	orders        *repository.OrderRepository
	products      *repository.ProductRepository
	customers     *repository.CustomerRepository
	notifications *repository.NotificationRepository
	payments      *integration.PaymentGateway
}

func NewOrderService(
	orders *repository.OrderRepository,
	products *repository.ProductRepository,
	customers *repository.CustomerRepository,
	notifications *repository.NotificationRepository,
	payments *integration.PaymentGateway,
) *OrderService {
	return &OrderService{orders: orders, products: products, customers: customers, notifications: notifications, payments: payments}
}

func (s *OrderService) Place(in PlaceOrderInput) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// R04
	if len(in.Items) == 0 {
		return model.Order{}, fail("validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > maxQuantityPerItem || seen[item.ProductID] {
			return model.Order{}, fail("validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
	}

	// R05
	if _, ok := s.customers.FindByID(in.CustomerID); !ok {
		return model.Order{}, fail("customer_not_found", "customer does not exist")
	}

	// R05, R06 and R07: check everything before changing any stock
	order := model.Order{CustomerID: in.CustomerID, Status: model.StatusPending, Coupon: in.Coupon}
	reserved := make([]model.Product, 0, len(in.Items))
	for _, item := range in.Items {
		product, ok := s.products.FindByID(item.ProductID)
		if !ok {
			return model.Order{}, fail("product_not_found", "product does not exist")
		}
		if product.Stock < item.Quantity {
			return model.Order{}, fail("insufficient_stock", "insufficient stock")
		}
		order.Items = append(order.Items, model.OrderItem{
			ProductID:      product.ID,
			Quantity:       item.Quantity,
			UnitPriceCents: product.PriceCents,
		})
		order.SubtotalCents += product.PriceCents * item.Quantity
		product.Stock -= item.Quantity
		reserved = append(reserved, product)
	}

	discount, err := s.discountFor(in.Coupon, in.CustomerID, order.SubtotalCents)
	if err != nil {
		return model.Order{}, err
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
	for _, product := range reserved {
		s.products.Update(product)
	}
	order = s.orders.Create(order)
	s.notify(order, "order_created") // R14
	return order, nil
}

// R08
func (s *OrderService) discountFor(coupon, customerID string, subtotal int) (int, error) {
	switch coupon {
	case "":
		return 0, nil
	case "WELCOME10":
		for _, previous := range s.orders.FindByCustomer(customerID) {
			if previous.Status != model.StatusCancelled {
				return 0, fail("coupon_not_eligible", "coupon valid only on the first order")
			}
		}
		return subtotal * 10 / 100, nil
	case "BIG20":
		if subtotal < bigOrderMinimumCents {
			return 0, fail("coupon_not_eligible", "subtotal below the coupon minimum")
		}
		return subtotal * 20 / 100, nil
	default:
		return 0, fail("invalid_coupon", "invalid coupon")
	}
}

func (s *OrderService) Get(id string) (model.Order, error) {
	order, ok := s.orders.FindByID(id)
	if !ok {
		return model.Order{}, fail("not_found", "order not found")
	}
	return order, nil
}

func (s *OrderService) Pay(id string) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, err := s.Get(id)
	if err != nil {
		return model.Order{}, err
	}
	// R11
	if order.Status != model.StatusPending {
		return model.Order{}, fail("invalid_transition", "only a pending order can be paid")
	}
	// R12
	if !s.payments.Charge(order.TotalCents) {
		return model.Order{}, fail("payment_declined", "payment declined")
	}
	order.Status = model.StatusPaid
	s.orders.Update(order)
	s.notify(order, "order_paid") // R14
	return order, nil
}

func (s *OrderService) Cancel(id string) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, err := s.Get(id)
	if err != nil {
		return model.Order{}, err
	}
	// R11
	if order.Status == model.StatusCancelled {
		return model.Order{}, fail("invalid_transition", "order already cancelled")
	}
	// R13
	for _, item := range order.Items {
		if product, ok := s.products.FindByID(item.ProductID); ok {
			product.Stock += item.Quantity
			s.products.Update(product)
		}
	}
	order.Status = model.StatusCancelled
	s.orders.Update(order)
	s.notify(order, "order_cancelled") // R14
	return order, nil
}

func (s *OrderService) notify(order model.Order, kind string) {
	s.notifications.Create(model.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: kind})
}
