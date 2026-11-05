// Catalog service: exclusive owner of products and stock.
// Nobody else reads this data directly. To learn about a product, you have to ask over HTTP.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type Line struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ReservedItem struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type Catalog struct {
	mu    sync.Mutex
	items map[string]Product
	seq   int
}

func main() {
	c := &Catalog{items: map[string]Product{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /products", c.create)
	mux.HandleFunc("GET /products/{id}", c.get)
	mux.HandleFunc("POST /reservations", c.reserve) // internal use, by orders
	mux.HandleFunc("POST /releases", c.release)     // internal use, by orders

	port := os.Getenv("PORT")
	log.Println("catalog on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func (c *Catalog) create(w http.ResponseWriter, r *http.Request) {
	var p Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, 400, "invalid_json", "invalid JSON")
		return
	}
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		writeError(w, 422, "validation", "product needs a name, a positive price and non-negative stock")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	p.ID = fmt.Sprintf("prod-%d", c.seq)
	c.items[p.ID] = p
	writeJSON(w, 201, p)
}

func (c *Catalog) get(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.items[r.PathValue("id")]
	if !ok {
		writeError(w, 404, "not_found", "product not found")
		return
	}
	writeJSON(w, 200, p)
}

// reserve is all or nothing (R06) and returns the current price (R07).
func (c *Catalog) reserve(w http.ResponseWriter, r *http.Request) {
	var lines []Line
	if err := json.NewDecoder(r.Body).Decode(&lines); err != nil {
		writeError(w, 400, "invalid_json", "invalid JSON")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, line := range lines {
		p, ok := c.items[line.ProductID]
		if !ok {
			writeError(w, 422, "product_not_found", "product does not exist") // R05
			return
		}
		if p.Stock < line.Quantity {
			writeError(w, 409, "insufficient_stock", "insufficient stock") // R06
			return
		}
	}
	out := make([]ReservedItem, len(lines))
	for i, line := range lines {
		p := c.items[line.ProductID]
		p.Stock -= line.Quantity
		c.items[p.ID] = p
		out[i] = ReservedItem{p.ID, line.Quantity, p.PriceCents}
	}
	writeJSON(w, 200, out)
}

// release returns stock (R13) and also undoes reservations (the saga's compensation).
func (c *Catalog) release(w http.ResponseWriter, r *http.Request) {
	var lines []Line
	if err := json.NewDecoder(r.Body).Decode(&lines); err != nil {
		writeError(w, 400, "invalid_json", "invalid JSON")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, line := range lines {
		if p, ok := c.items[line.ProductID]; ok {
			p.Stock += line.Quantity
			c.items[p.ID] = p
		}
	}
	w.WriteHeader(204)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}
