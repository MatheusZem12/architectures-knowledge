// Package web has what all features share to speak HTTP.
package web

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func Fail(code, message string) *Error { return &Error{Code: code, Message: message} }

func Decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		JSON(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return false
	}
	return true
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Reply(w http.ResponseWriter, status int) func(any, error) {
	return func(value any, err error) {
		if err == nil {
			JSON(w, status, value)
			return
		}
		var businessErr *Error
		if errors.As(err, &businessErr) {
			JSON(w, statusFor(businessErr.Code), map[string]string{"code": businessErr.Code, "error": businessErr.Message})
			return
		}
		JSON(w, 500, map[string]string{"code": "internal", "error": "internal error"})
	}
}

func statusFor(code string) int {
	switch code {
	case "not_found":
		return 404
	case "payment_declined":
		return 402
	case "email_taken", "insufficient_stock", "invalid_transition":
		return 409
	default:
		return 422
	}
}
