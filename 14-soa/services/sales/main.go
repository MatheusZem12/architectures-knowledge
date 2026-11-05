// Sales service: a coarse service that owns both the catalog and orders.
//
// Two SOA habits are visible here. It reads the customers table directly, because the database
// is shared. And it reaches billing through the bus, naming an operation rather than a service.
package main

import (
	"encoding/json"
	"strings"
	"sync"

	"soa/canonical"
)

var (
	db  = canonical.NewDB()
	bus = canonical.NewBus()
	mu  sync.Mutex // the shared database offers no transaction here, so this keeps sales consistent
)

func main() {
	canonical.Serve("sales", map[string]canonical.Handler{
		"CreateProduct": createProduct,
		"GetProduct":    getProduct,
		"PlaceOrder":    placeOrder,
		"GetOrder":      getOrder,
		"PayOrder":      payOrder,
		"CancelOrder":   cancelOrder,
	})
}

func createProduct(payload json.RawMessage) (any, error) {
	var in canonical.Product
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	// R01
	if strings.TrimSpace(in.Name) == "" || in.PriceCents <= 0 || in.Stock < 0 {
		return nil, canonical.Fail(422, "validation", "product needs a name, a positive price and non-negative stock")
	}
	var err error
	if in.ID, err = db.NextID("prod"); err != nil {
		return nil, err
	}
	return in, db.Put("products", in.ID, in)
}

func getProduct(payload json.RawMessage) (any, error) {
	var in canonical.ByID
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	var product canonical.Product
	found, err := db.Get("products", in.ID, &product)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, canonical.Fail(404, "not_found", "product not found")
	}
	return product, nil
}

type placeInput struct {
	CustomerID string `json:"customer_id"`
	Coupon     string `json:"coupon"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

func placeOrder(payload json.RawMessage) (any, error) {
	var in placeInput
	if err := canonical.Decode(payload, &in); err != nil {
		return nil, err
	}
	// R04
	if len(in.Items) == 0 {
		return nil, canonical.Fail(422, "validation", "order needs at least one item")
	}
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return nil, canonical.Fail(422, "validation", "item with invalid quantity or repeated")
		}
		seen[item.ProductID] = true
	}

	mu.Lock()
	defer mu.Unlock()

	// R05: straight to the customers table, skipping the customer service. Quick, and coupling:
	// if that team changes the table, this breaks.
	var customer canonical.Customer
	found, err := db.Get("customers", in.CustomerID, &customer)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, canonical.Fail(422, "customer_not_found", "customer does not exist")
	}

	// R05, R06 and R07: check everything before changing any stock
	order := canonical.Order{CustomerID: in.CustomerID, Status: "pending", Coupon: in.Coupon}
	products := make([]canonical.Product, 0, len(in.Items))
	for _, item := range in.Items {
		var product canonical.Product
		found, err := db.Get("products", item.ProductID, &product)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, canonical.Fail(422, "product_not_found", "product does not exist")
		}
		if product.Stock < item.Quantity {
			return nil, canonical.Fail(409, "insufficient_stock", "insufficient stock")
		}
		order.Items = append(order.Items, canonical.OrderItem{ProductID: product.ID, Quantity: item.Quantity, UnitPriceCents: product.PriceCents})
		order.SubtotalCents += product.PriceCents * item.Quantity
		product.Stock -= item.Quantity
		products = append(products, product)
	}

	isFirstOrder := true
	if in.Coupon == "WELCOME10" {
		if isFirstOrder, err = firstOrder(in.CustomerID); err != nil {
			return nil, err
		}
	}
	discount, err := discountFor(in.Coupon, order.SubtotalCents, isFirstOrder)
	if err != nil {
		return nil, err
	}
	order.DiscountCents = discount
	order.ShippingCents = shippingFor(order.SubtotalCents - discount)
	// R10
	order.TotalCents = order.SubtotalCents - order.DiscountCents + order.ShippingCents

	// R06: only save the stock once nothing else can fail
	for _, product := range products {
		if err := db.Put("products", product.ID, product); err != nil {
			return nil, err
		}
	}
	if order.ID, err = db.NextID("ord"); err != nil {
		return nil, err
	}
	if err := db.Put("orders", order.ID, order); err != nil {
		return nil, err
	}
	return order, notify(order, "order_created")
}

func firstOrder(customerID string) (bool, error) {
	var previous []canonical.Order
	if err := db.List("orders", "customer_id", customerID, &previous); err != nil {
		return false, err
	}
	for _, order := range previous {
		if order.Status != "cancelled" {
			return false, nil
		}
	}
	return true, nil
}

// notify waits for billing to answer: in classic SOA, calls through the bus are usually synchronous.
func notify(order canonical.Order, kind string) error {
	return bus.Call("SendNotification", canonical.Notification{CustomerID: order.CustomerID, OrderID: order.ID, Type: kind}, nil)
}

func findOrder(payload json.RawMessage) (canonical.Order, error) {
	var in canonical.ByID
	if err := canonical.Decode(payload, &in); err != nil {
		return canonical.Order{}, err
	}
	var order canonical.Order
	found, err := db.Get("orders", in.ID, &order)
	if err != nil {
		return canonical.Order{}, err
	}
	if !found {
		return canonical.Order{}, canonical.Fail(404, "not_found", "order not found")
	}
	return order, nil
}

func getOrder(payload json.RawMessage) (any, error) {
	return findOrder(payload)
}

func payOrder(payload json.RawMessage) (any, error) {
	mu.Lock()
	defer mu.Unlock()
	order, err := findOrder(payload)
	if err != nil {
		return nil, err
	}
	// R11
	if order.Status != "pending" {
		return nil, canonical.Fail(409, "invalid_transition", "only a pending order can be paid")
	}
	if err := bus.Call("ChargePayment", canonical.Charge{AmountCents: order.TotalCents}, nil); err != nil {
		return nil, err
	}
	order.Status = "paid"
	if err := db.Put("orders", order.ID, order); err != nil {
		return nil, err
	}
	return order, notify(order, "order_paid")
}

func cancelOrder(payload json.RawMessage) (any, error) {
	mu.Lock()
	defer mu.Unlock()
	order, err := findOrder(payload)
	if err != nil {
		return nil, err
	}
	// R11
	if order.Status == "cancelled" {
		return nil, canonical.Fail(409, "invalid_transition", "order already cancelled")
	}
	// R13
	for _, item := range order.Items {
		var product canonical.Product
		found, err := db.Get("products", item.ProductID, &product)
		if err != nil {
			return nil, err
		}
		if found {
			product.Stock += item.Quantity
			if err := db.Put("products", product.ID, product); err != nil {
				return nil, err
			}
		}
	}
	order.Status = "cancelled"
	if err := db.Put("orders", order.ID, order); err != nil {
		return nil, err
	}
	return order, notify(order, "order_cancelled")
}
