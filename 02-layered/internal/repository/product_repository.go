// Package repository is the persistence layer. It only stores and fetches: it decides nothing.
package repository

import (
	"fmt"
	"sync"

	"layered/internal/model"
)

type ProductRepository struct {
	mu    sync.RWMutex
	items map[string]model.Product
	seq   int
}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{items: map[string]model.Product{}}
}

func (r *ProductRepository) Create(p model.Product) model.Product {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	p.ID = fmt.Sprintf("prod-%d", r.seq)
	r.items[p.ID] = p
	return p
}

func (r *ProductRepository) FindByID(id string) (model.Product, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.items[id]
	return p, ok
}

func (r *ProductRepository) Update(p model.Product) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[p.ID] = p
}
