// Package entities is the innermost ring: the rules that would hold even without this system.
// It imports nothing from the project, and only uses the standard library for the basics.
package entities

const (
	CodeValidation        = "validation"
	CodeNotFound          = "not_found"
	CodeEmailTaken        = "email_taken"
	CodeCustomerNotFound  = "customer_not_found"
	CodeProductNotFound   = "product_not_found"
	CodeInsufficientStock = "insufficient_stock"
	CodeInvalidCoupon     = "invalid_coupon"
	CodeCouponNotEligible = "coupon_not_eligible"
	CodeInvalidTransition = "invalid_transition"
	CodePaymentDeclined   = "payment_declined"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func Fail(code, message string) *Error { return &Error{Code: code, Message: message} }
