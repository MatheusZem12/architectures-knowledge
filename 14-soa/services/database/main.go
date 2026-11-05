// The shared database, standing in for the one Oracle or SQL Server the whole company uses.
// It is a small process with tables of JSON rows, and it knows no business rule.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

type Database struct {
	mu     sync.RWMutex
	tables map[string]map[string]json.RawMessage
	seq    int
}

func main() {
	db := &Database{tables: map[string]map[string]json.RawMessage{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("GET /rows/{table}/{id}", db.get)
	mux.HandleFunc("PUT /rows/{table}/{id}", db.put)
	mux.HandleFunc("GET /rows/{table}", db.list)
	mux.HandleFunc("POST /ids/{prefix}", db.nextID)

	port := os.Getenv("PORT")
	log.Println("database on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func (d *Database) get(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	row, ok := d.tables[r.PathValue("table")][r.PathValue("id")]
	if !ok {
		w.WriteHeader(404)
		return
	}
	w.Write(row)
}

func (d *Database) put(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	table := r.PathValue("table")
	if d.tables[table] == nil {
		d.tables[table] = map[string]json.RawMessage{}
	}
	d.tables[table][r.PathValue("id")] = raw
	w.WriteHeader(204)
}

// list filters on one field, the way a WHERE clause would.
func (d *Database) list(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []json.RawMessage{}
	for _, row := range d.tables[r.PathValue("table")] {
		var fields map[string]any
		if err := json.Unmarshal(row, &fields); err != nil {
			continue
		}
		matches := true
		for field, values := range r.URL.Query() {
			if fmt.Sprint(fields[field]) != values[0] {
				matches = false
			}
		}
		if matches {
			out = append(out, row)
		}
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (d *Database) nextID(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	_ = json.NewEncoder(w).Encode(map[string]string{"id": fmt.Sprintf("%s-%d", r.PathValue("prefix"), d.seq)})
}
