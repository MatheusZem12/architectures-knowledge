// Package events is the only thing the modules have in common: the facts and requests they
// exchange. Reading this file top to bottom is reading the system's flow.
//
// RequestID links a chain of events to the HTTP request waiting for its outcome.
package events

type Line struct {
	ProductID string
	Quantity  int
}

type Item struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

// ---------- placing an order ----------

type OrderRequested struct {
	RequestID  string
	CustomerID string
	Lines      []Line
	Coupon     string
}

type CustomerConfirmed struct {
	RequestID  string
	CustomerID string
	Lines      []Line
	Coupon     string
}

type StockReserved struct {
	RequestID  string
	CustomerID string
	Items      []Item
	Coupon     string
}

// ReservationRejected asks the catalog to undo a reservation: it is the saga's compensation.
type ReservationRejected struct {
	RequestID string
	Lines     []Line
	Code      string
	Message   string
}

type OrderRejected struct {
	RequestID string
	Code      string
	Message   string
}

type OrderPlaced struct {
	OrderID    string
	CustomerID string
}

// ---------- paying an order ----------

type PayRequested struct {
	RequestID string
	OrderID   string
}

type ChargeRequested struct {
	RequestID   string
	OrderID     string
	AmountCents int
}

type PaymentApproved struct {
	RequestID string
	OrderID   string
}

type PaymentDeclined struct {
	RequestID string
	OrderID   string
}

type OrderPaid struct {
	OrderID    string
	CustomerID string
}

// ---------- cancelling an order ----------

type CancelRequested struct {
	RequestID string
	OrderID   string
}

type ReleaseRequested struct {
	RequestID string
	OrderID   string
	Lines     []Line
}

type StockReleased struct {
	RequestID string
	OrderID   string
}

type OrderCancelled struct {
	OrderID    string
	CustomerID string
}
