// main creates the broker and starts the modules. Nothing else connects them: from here on,
// they only talk through events. Run with LOG_EVENTS=1 to watch the events go by.
package main

import (
	"log"
	"net/http"
	"os"

	"eventdriven/modules/catalog"
	"eventdriven/modules/customers"
	"eventdriven/modules/notifications"
	"eventdriven/modules/orders"
	"eventdriven/modules/payments"
	"eventdriven/platform/broker"
)

func main() {
	bus := broker.New(os.Getenv("LOG_EVENTS") == "1")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	catalog.New(bus).Routes(mux)
	customers.New(bus).Routes(mux)
	orders.New(bus).Routes(mux)
	notifications.New(bus).Routes(mux)
	payments.New(bus)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("09-event-driven on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
