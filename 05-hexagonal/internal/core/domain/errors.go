// Package domain is the heart of the system: the store's rules, in plain Go.
// Nothing here knows that HTTP, JSON or databases exist.
package domain

type Code string

const (
	CodeValidation        Code = "validation"
	CodeNotFound          Code = "not_found"
	CodeEmailTaken        Code = "email_taken"
	CodeCustomerNotFound  Code = "customer_not_found"
	CodeProductNotFound   Code = "product_not_found"
	CodeInsufficientStock Code = "insufficient_stock"
	CodeInvalidCoupon     Code = "invalid_coupon"
	CodeCouponNotEligible Code = "coupon_not_eligible"
	CodeInvalidTransition Code = "invalid_transition"
	CodePaymentDeclined   Code = "payment_declined"
)

type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string { return e.Message }

// Fail creates an error in the domain's language. The application layer uses it too.
func Fail(code Code, message string) *Error { return &Error{Code: code, Message: message} }
