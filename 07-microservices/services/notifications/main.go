// Notifications service: stores the notices sent by the orders service.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

type Notification struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

type Inbox struct {
	mu    sync.RWMutex
	items map[string][]Notification
}

func main() {
	inbox := &Inbox{items: map[string][]Notification{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /notifications", inbox.add)
	mux.HandleFunc("GET /customers/{id}/notifications", inbox.list)

	port := os.Getenv("PORT")
	log.Println("notifications on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// R14
func (i *Inbox) add(w http.ResponseWriter, r *http.Request) {
	var n Notification
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeJSON(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.items[n.CustomerID] = append(i.items[n.CustomerID], n)
	w.WriteHeader(204)
}

func (i *Inbox) list(w http.ResponseWriter, r *http.Request) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	writeJSON(w, 200, append([]Notification{}, i.items[r.PathValue("id")]...))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
