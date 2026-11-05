// The gateway is the only entry point. It has no business rules: it only forwards
// and answers 503 when the target service is down.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	services := map[string]string{
		"catalog":       env("CATALOG_URL", "http://localhost:8081"),
		"customers":     env("CUSTOMERS_URL", "http://localhost:8082"),
		"orders":        env("ORDERS_URL", "http://localhost:8083"),
		"payments":      env("PAYMENTS_URL", "http://localhost:8084"),
		"notifications": env("NOTIFICATIONS_URL", "http://localhost:8085"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { health(w, services) })
	mux.Handle("/products", proxy(services["catalog"]))
	mux.Handle("/products/", proxy(services["catalog"]))
	mux.Handle("/customers", proxy(services["customers"]))
	mux.Handle("/customers/", proxy(services["customers"]))
	mux.Handle("GET /customers/{id}/notifications", proxy(services["notifications"]))
	mux.Handle("/orders", proxy(services["orders"]))
	mux.Handle("/orders/", proxy(services["orders"]))

	port := env("PORT", "8080")
	log.Println("gateway on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func proxy(target string) http.Handler {
	address, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}
	p := httputil.NewSingleHostReverseProxy(address)
	p.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		writeJSON(w, 503, map[string]string{"code": "service_unavailable", "error": "service unavailable"})
	}
	return p
}

// health only answers 200 when all services answer.
func health(w http.ResponseWriter, services map[string]string) {
	client := http.Client{Timeout: time.Second}
	var down []string
	for name, address := range services {
		res, err := client.Get(address + "/health")
		if err != nil {
			down = append(down, name)
			continue
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			down = append(down, name)
		}
	}
	if len(down) > 0 {
		writeJSON(w, 503, map[string]string{"code": "service_unavailable", "error": "down: " + strings.Join(down, ", ")})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
