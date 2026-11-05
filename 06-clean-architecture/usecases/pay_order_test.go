package usecases

import (
	"errors"
	"testing"

	"cleanarch/entities"
)

// Thanks to the small interfaces, the fake implements only the methods PayOrder really uses.
type fakeOrders struct{ order entities.Order }

func (f *fakeOrders) FindOrder(string) (entities.Order, bool) { return f.order, true }
func (f *fakeOrders) SaveOrder(o entities.Order)              { f.order = o }

type fakeNotifications struct{ sent []string }

func (f *fakeNotifications) AddNotification(n entities.Notification) { f.sent = append(f.sent, n.Type) }

type fakePayments struct{ approve bool }

func (f fakePayments) Charge(int) bool { return f.approve }

type noTransaction struct{}

func (noTransaction) Atomic(fn func() error) error { return fn() }

func TestPayOrder(t *testing.T) {
	t.Run("approved: marks as paid and notifies", func(t *testing.T) {
		orders := &fakeOrders{order: entities.Order{ID: "1", Status: entities.StatusPending}}
		notifications := &fakeNotifications{}

		out, err := NewPayOrder(orders, notifications, fakePayments{approve: true}, noTransaction{}).Execute("1")

		if err != nil || out.Status != entities.StatusPaid || len(notifications.sent) != 1 {
			t.Fatalf("out = %+v, err = %v, notifications = %v", out, err, notifications.sent)
		}
	})

	t.Run("declined: stays pending and does not notify", func(t *testing.T) {
		orders := &fakeOrders{order: entities.Order{ID: "1", Status: entities.StatusPending}}
		notifications := &fakeNotifications{}

		_, err := NewPayOrder(orders, notifications, fakePayments{approve: false}, noTransaction{}).Execute("1")

		var businessErr *entities.Error
		if !errors.As(err, &businessErr) || businessErr.Code != entities.CodePaymentDeclined {
			t.Fatalf("err = %v", err)
		}
		if orders.order.Status != entities.StatusPending || len(notifications.sent) != 0 {
			t.Fatalf("status = %s, notifications = %v", orders.order.Status, notifications.sent)
		}
	})
}
