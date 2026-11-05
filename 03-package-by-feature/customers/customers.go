// Package customers gathers everything about customers.
package customers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"byfeature/web"
)

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Store struct {
	mu    sync.RWMutex
	items map[string]Customer
	seq   int
}

func NewStore() *Store { return &Store{items: map[string]Customer{}} }

func (s *Store) Create(c Customer) (Customer, error) {
	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return Customer{}, web.Fail("validation", "customer needs a name and a valid email")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// R03
	for _, existing := range s.items {
		if strings.EqualFold(existing.Email, c.Email) {
			return Customer{}, web.Fail("email_taken", "email already registered")
		}
	}
	s.seq++
	c.ID = fmt.Sprintf("cust-%d", s.seq)
	s.items[c.ID] = c
	return c, nil
}

func (s *Store) Find(id string) (Customer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[id]
	return c, ok
}

func Register(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in Customer
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(store.Create(in))
		}
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, ok := store.Find(r.PathValue("id"))
		if !ok {
			web.Reply(w, 404)(c, web.Fail("not_found", "customer not found"))
			return
		}
		web.JSON(w, 200, c)
	})
}
