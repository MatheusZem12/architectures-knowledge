// Package customers is the customers module.
package customers

import (
	"net/http"
	"strings"

	"modular/modules/customers/internal/store"
	"modular/platform/web"
)

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Module struct {
	store *store.Store
}

func New() *Module { return &Module{store: store.New()} }

func (m *Module) Create(c Customer) (Customer, error) {
	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return Customer{}, web.Fail("validation", "customer needs a name and a valid email")
	}
	// R03: the check and the write happen together, inside the store
	record, ok := m.store.InsertIfEmailFree(store.Record{Name: c.Name, Email: c.Email})
	if !ok {
		return Customer{}, web.Fail("email_taken", "email already registered")
	}
	return Customer{ID: record.ID, Name: record.Name, Email: record.Email}, nil
}

func (m *Module) Get(id string) (Customer, error) {
	record, ok := m.store.Find(id)
	if !ok {
		return Customer{}, web.Fail("not_found", "customer not found")
	}
	return Customer{ID: record.ID, Name: record.Name, Email: record.Email}, nil
}

// Exists is all the orders module needs to know about a customer (R05).
func (m *Module) Exists(id string) bool {
	_, ok := m.store.Find(id)
	return ok
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /customers", func(w http.ResponseWriter, r *http.Request) {
		var in Customer
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.Create(in))
		}
	})
	mux.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.Get(r.PathValue("id")))
	})
}
