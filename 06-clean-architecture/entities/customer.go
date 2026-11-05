package entities

import "strings"

type Customer struct {
	ID    string
	Name  string
	Email string
}

func NewCustomer(name, email string) (Customer, error) {
	// R02
	at := strings.Index(email, "@")
	if strings.TrimSpace(name) == "" || at <= 0 || at == len(email)-1 {
		return Customer{}, Fail(CodeValidation, "customer needs a name and a valid email")
	}
	return Customer{Name: name, Email: email}, nil
}
