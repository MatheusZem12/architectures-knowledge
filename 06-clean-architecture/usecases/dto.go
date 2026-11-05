package usecases

import "cleanarch/entities"

// The output DTOs keep entities from leaking to the outer rings.

type ProductOutput struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

type CustomerOutput struct {
	ID    string
	Name  string
	Email string
}

type OrderItemOutput struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

type OrderOutput struct {
	ID            string
	CustomerID    string
	Status        string
	Items         []OrderItemOutput
	Coupon        string
	SubtotalCents int
	DiscountCents int
	ShippingCents int
	TotalCents    int
}

type NotificationOutput struct {
	CustomerID string
	OrderID    string
	Type       string
}

func productOutput(p entities.Product) ProductOutput { return ProductOutput(p) }

func customerOutput(c entities.Customer) CustomerOutput { return CustomerOutput(c) }

func orderOutput(o entities.Order) OrderOutput {
	items := make([]OrderItemOutput, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemOutput(item)
	}
	return OrderOutput{o.ID, o.CustomerID, o.Status, items, o.Coupon, o.SubtotalCents, o.DiscountCents, o.ShippingCents, o.TotalCents}
}
