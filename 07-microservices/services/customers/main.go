// Customers service: exclusive owner of customer data.
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

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Customers struct {
	mu    sync.Mutex
	items map[string]Customer
	seq   int
}

func main() {
	c := &Customers{items: map[string]Customer{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /customers", c.create)
	mux.HandleFunc("GET /customers/{id}", c.get)

	port := os.Getenv("PORT")
	log.Println("customers on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func (c *Customers) create(w http.ResponseWriter, r *http.Request) {
	var in Customer
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", "invalid JSON")
		return
	}
	// R02
	at := strings.Index(in.Email, "@")
	if strings.TrimSpace(in.Name) == "" || at <= 0 || at == len(in.Email)-1 {
		writeError(w, 422, "validation", "customer needs a name and a valid email")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// R03
	for _, existing := range c.items {
		if strings.EqualFold(existing.Email, in.Email) {
			writeError(w, 409, "email_taken", "email already registered")
			return
		}
	}
	c.seq++
	in.ID = fmt.Sprintf("cust-%d", c.seq)
	c.items[in.ID] = in
	writeJSON(w, 201, in)
}

func (c *Customers) get(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	customer, ok := c.items[r.PathValue("id")]
	if !ok {
		writeError(w, 404, "not_found", "customer not found")
		return
	}
	writeJSON(w, 200, customer)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}
