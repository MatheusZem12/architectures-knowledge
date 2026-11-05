// Package payments reacts to charge requests. Today it simulates an external gateway.
package payments

import (
	"eventdriven/events"
	"eventdriven/platform/broker"
)

const maxChargeCents = 1_000_000

type Module struct {
	bus *broker.Broker
}

func New(bus *broker.Broker) *Module {
	m := &Module{bus: bus}
	bus.Subscribe("payments", m.handle)
	return m
}

func (m *Module) handle(event any) error {
	e, ok := event.(events.ChargeRequested)
	if !ok {
		return nil
	}
	// R12
	if e.AmountCents > maxChargeCents {
		m.bus.Publish(events.PaymentDeclined{RequestID: e.RequestID, OrderID: e.OrderID})
		return nil
	}
	m.bus.Publish(events.PaymentApproved{RequestID: e.RequestID, OrderID: e.OrderID})
	return nil
}
