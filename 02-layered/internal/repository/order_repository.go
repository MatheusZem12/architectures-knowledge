package repository

import (
	"fmt"
	"sync"

	"layered/internal/model"
)

type OrderRepository struct {
	mu    sync.RWMutex
	items map[string]model.Order
	seq   int
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{items: map[string]model.Order{}}
}

func (r *OrderRepository) Create(o model.Order) model.Order {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	o.ID = fmt.Sprintf("ord-%d", r.seq)
	r.items[o.ID] = o
	return o
}

func (r *OrderRepository) FindByID(id string) (model.Order, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.items[id]
	return o, ok
}

func (r *OrderRepository) Update(o model.Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[o.ID] = o
}

func (r *OrderRepository) FindByCustomer(customerID string) []model.Order {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []model.Order
	for _, o := range r.items {
		if o.CustomerID == customerID {
			out = append(out, o)
		}
	}
	return out
}
