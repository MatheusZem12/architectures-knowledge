// Package products gathers everything about products: type, rules, storage and HTTP.
//
// To understand products, just open this folder. But notice: everything here is exported.
// Any other package can use Store and Product freely, and orders does.
package products

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"byfeature/web"
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string]Product
	seq   int
}

func NewStore() *Store { return &Store{items: map[string]Product{}} }

func (s *Store) Create(p Product) (Product, error) {
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		return Product{}, web.Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	p.ID = fmt.Sprintf("prod-%d", s.seq)
	s.items[p.ID] = p
	return p, nil
}

func (s *Store) Find(id string) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	return p, ok
}

func (s *Store) Save(p Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[p.ID] = p
}

func Register(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in Product
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(store.Create(in))
		}
	})
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := store.Find(r.PathValue("id"))
		if !ok {
			web.Reply(w, 404)(p, web.Fail("not_found", "product not found"))
			return
		}
		web.JSON(w, 200, p)
	})
}
