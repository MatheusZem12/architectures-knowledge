// Package catalog is the products and stock module.
//
// This file is the module's public API: it is all the other modules can see.
// Storage lives in internal/, and the Go compiler itself prevents anyone outside
// the catalog folder from importing it.
package catalog

import (
	"net/http"
	"strings"

	"modular/modules/catalog/internal/store"
	"modular/platform/web"
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

// Line is a reservation request: how much of which product.
type Line struct {
	ProductID string
	Quantity  int
}

// Reserved is a completed reservation, with the current price (R07).
type Reserved struct {
	ProductID      string
	Quantity       int
	UnitPriceCents int
}

type Module struct {
	store *store.Store
}

func New() *Module { return &Module{store: store.New()} }

func (m *Module) Create(p Product) (Product, error) {
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		return Product{}, web.Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	return fromRecord(m.store.Insert(store.Record{Name: p.Name, PriceCents: p.PriceCents, Stock: p.Stock})), nil
}

func (m *Module) Get(id string) (Product, error) {
	record, ok := m.store.Find(id)
	if !ok {
		return Product{}, web.Fail("not_found", "product not found")
	}
	return fromRecord(record), nil
}

// Reserve deducts stock for all lines, or for none (R06).
func (m *Module) Reserve(lines []Line) ([]Reserved, error) {
	wanted := make([]store.Line, len(lines))
	for i, line := range lines {
		wanted[i] = store.Line{ProductID: line.ProductID, Quantity: line.Quantity}
	}
	records, err := m.store.ReserveAll(wanted)
	switch err {
	case nil:
	case store.ErrNotFound:
		return nil, web.Fail("product_not_found", "product does not exist") // R05
	default:
		return nil, web.Fail("insufficient_stock", "insufficient stock") // R06
	}
	out := make([]Reserved, len(lines))
	for i, record := range records {
		out[i] = Reserved{ProductID: record.ID, Quantity: lines[i].Quantity, UnitPriceCents: record.PriceCents}
	}
	return out, nil
}

// Release returns what had been reserved to stock (R13).
func (m *Module) Release(lines []Line) {
	for _, line := range lines {
		m.store.AddStock(line.ProductID, line.Quantity)
	}
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var in Product
		if web.Decode(w, r, &in) {
			web.Reply(w, 201)(m.Create(in))
		}
	})
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		web.Reply(w, 200)(m.Get(r.PathValue("id")))
	})
}

func fromRecord(r store.Record) Product {
	return Product{ID: r.ID, Name: r.Name, PriceCents: r.PriceCents, Stock: r.Stock}
}
