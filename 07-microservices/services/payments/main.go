// Payments service: simulates an external gateway.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

const maxChargeCents = 1_000_000

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /charges", charge)

	port := os.Getenv("PORT")
	log.Println("payments on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func charge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AmountCents int `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"code": "invalid_json", "error": "invalid JSON"})
		return
	}
	// R12
	if in.AmountCents > maxChargeCents {
		writeJSON(w, 402, map[string]string{"code": "payment_declined", "error": "payment declined"})
		return
	}
	writeJSON(w, 200, map[string]bool{"approved": true})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
