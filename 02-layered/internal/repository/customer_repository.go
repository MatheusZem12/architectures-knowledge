package repository

import (
	"fmt"
	"strings"
	"sync"

	"layered/internal/model"
)

type CustomerRepository struct {
	mu    sync.RWMutex
	items map[string]model.Customer
	seq   int
}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{items: map[string]model.Customer{}}
}

func (r *CustomerRepository) Create(c model.Customer) model.Customer {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	c.ID = fmt.Sprintf("cust-%d", r.seq)
	r.items[c.ID] = c
	return c
}

func (r *CustomerRepository) FindByID(id string) (model.Customer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.items[id]
	return c, ok
}

func (r *CustomerRepository) ExistsByEmail(email string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.items {
		if strings.EqualFold(c.Email, email) {
			return true
		}
	}
	return false
}
