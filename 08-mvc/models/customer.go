package models

import "strings"

type Customer struct {
	ID    string
	Name  string
	Email string
}

func (db *DB) CreateCustomer(name, email string) (Customer, error) {
	// R02
	at := strings.Index(email, "@")
	if strings.TrimSpace(name) == "" || at <= 0 || at == len(email)-1 {
		return Customer{}, Fail("validation", "customer needs a name and a valid email")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	// R03
	for _, existing := range db.customers {
		if strings.EqualFold(existing.Email, email) {
			return Customer{}, Fail("email_taken", "email already registered")
		}
	}
	customer := Customer{ID: db.nextID("cust"), Name: name, Email: email}
	db.customers[customer.ID] = customer
	return customer, nil
}

func (db *DB) FindCustomer(id string) (Customer, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	customer, ok := db.customers[id]
	if !ok {
		return Customer{}, Fail("not_found", "customer not found")
	}
	return customer, nil
}
