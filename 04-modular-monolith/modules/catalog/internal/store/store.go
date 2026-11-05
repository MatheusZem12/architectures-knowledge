// Package store is private to the catalog module: no other module can import it.
// In a real database, these would be the tables (or schema) exclusive to the catalog.
package store

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotFound = errors.New("product not found")
	ErrNoStock  = errors.New("insufficient stock")
)

type Record struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

type Line struct {
	ProductID string
	Quantity  int
}

type Store struct {
	mu    sync.Mutex
	items map[string]Record
	seq   int
}

func New() *Store { return &Store{items: map[string]Record{}} }

func (s *Store) Insert(r Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	r.ID = fmt.Sprintf("prod-%d", s.seq)
	s.items[r.ID] = r
	return r
}

func (s *Store) Find(id string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	return r, ok
}

// ReserveAll checks every line before changing any of them: all or nothing.
func (s *Store) ReserveAll(lines []Line) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, line := range lines {
		r, ok := s.items[line.ProductID]
		if !ok {
			return nil, ErrNotFound
		}
		if r.Stock < line.Quantity {
			return nil, ErrNoStock
		}
	}
	out := make([]Record, len(lines))
	for i, line := range lines {
		r := s.items[line.ProductID]
		r.Stock -= line.Quantity
		s.items[line.ProductID] = r
		out[i] = r
	}
	return out, nil
}

func (s *Store) AddStock(id string, quantity int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.items[id]; ok {
		r.Stock += quantity
		s.items[id] = r
	}
}
