// Package kernel is the small, stable core. It knows how an order works, and it knows that plugins
// exist. It does not know which plugins exist: no file in this folder imports one.
package kernel

import "net/http"

// Plugin is the only thing an extension has to implement.
type Plugin interface {
	Name() string
	Install(registry *Registry)
}

// Coupon, ShippingPolicy, PaymentProvider and Listener are the extension points: the only ways in.

type Coupon interface {
	Discount(subtotalCents int, isFirstOrder bool) (int, error)
}

type ShippingPolicy func(subtotalAfterDiscountCents int) int

type PaymentProvider func(amountCents int) bool

type OrderEvent struct {
	CustomerID string
	OrderID    string
	Type       string
}

type Listener func(OrderEvent)

// Registry is what a plugin receives when it is installed.
type Registry struct {
	kernel *Kernel
}

func (r *Registry) AddCoupon(code string, coupon Coupon) { r.kernel.coupons[code] = coupon }

func (r *Registry) SetShipping(policy ShippingPolicy) { r.kernel.shipping = policy }

func (r *Registry) SetPayment(provider PaymentProvider) { r.kernel.payment = provider }

func (r *Registry) OnOrderEvent(listener Listener) {
	r.kernel.listeners = append(r.kernel.listeners, listener)
}

// AddRoute lets a plugin bring its own endpoints.
func (r *Registry) AddRoute(pattern string, handler http.HandlerFunc) {
	r.kernel.mux.HandleFunc(pattern, handler)
}
