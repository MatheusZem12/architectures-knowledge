package write

import (
	"errors"
	"testing"

	"cqrs/events"
)

type spy struct{ published []any }

func (s *spy) Publish(event any) { s.published = append(s.published, event) }

type declineAll struct{}

func (declineAll) Charge(int) bool { return false }

// The write side is tested by what it decides and publishes. There are no views to set up.
func TestPlaceOrderPublishesTheFacts(t *testing.T) {
	published := &spy{}
	handler := NewHandler(published, FakeGateway{})
	product, _ := handler.CreateProduct(CreateProduct{"Keyboard", 30_000, 5})
	customer, _ := handler.RegisterCustomer(RegisterCustomer{"Ana", "ana@example.com"})
	published.published = nil

	_, err := handler.PlaceOrder(PlaceOrder{customer, []OrderLine{{product, 2}}, "WELCOME10"})
	if err != nil {
		t.Fatal(err)
	}

	if len(published.published) != 2 {
		t.Fatalf("published = %+v", published.published)
	}
	if got := published.published[0].(events.StockChanged); got.Stock != 3 {
		t.Fatalf("stock event = %+v", got)
	}
	if got := published.published[1].(events.OrderPlaced); got.DiscountCents != 6_000 || got.TotalCents != 54_000 {
		t.Fatalf("order event = %+v", got)
	}
}

func TestRefusedCommandPublishesNothing(t *testing.T) {
	published := &spy{}
	handler := NewHandler(published, declineAll{})
	product, _ := handler.CreateProduct(CreateProduct{"Keyboard", 100, 1})
	customer, _ := handler.RegisterCustomer(RegisterCustomer{"Ana", "ana@example.com"})
	id, err := handler.PlaceOrder(PlaceOrder{CustomerID: customer, Lines: []OrderLine{{product, 1}}})
	if err != nil {
		t.Fatal(err)
	}
	published.published = nil

	err = handler.PayOrder(id)

	var refusal *Error
	if !errors.As(err, &refusal) || refusal.Code != "payment_declined" {
		t.Fatalf("error = %v", err)
	}
	if len(published.published) != 0 {
		t.Fatalf("published = %+v", published.published)
	}
}
