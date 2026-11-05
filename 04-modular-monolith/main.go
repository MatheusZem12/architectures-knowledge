// main is the only place that knows all modules. This is where they meet.
package main

import (
	"log"
	"net/http"
	"os"

	"modular/modules/catalog"
	"modular/modules/customers"
	"modular/modules/notifications"
	"modular/modules/orders"
	"modular/modules/payments"
	"modular/platform/events"
)

func main() {
	bus := events.NewBus()

	catalogModule := catalog.New()
	customersModule := customers.New()
	paymentsModule := payments.New()
	ordersModule := orders.New(catalogModule, customersModule, paymentsModule, bus)
	notificationsModule := notifications.New(bus)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	catalogModule.Routes(mux)
	customersModule.Routes(mux)
	ordersModule.Routes(mux)
	notificationsModule.Routes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("04-modular-monolith on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
