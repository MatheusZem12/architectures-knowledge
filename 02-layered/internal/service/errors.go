// Package service is the business layer: every decision of the store lives here.
package service

// Error is a business error. The handler translates Code into an HTTP status.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func fail(code, message string) *Error { return &Error{Code: code, Message: message} }
