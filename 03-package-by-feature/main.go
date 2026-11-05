package main

import (
	"log"
	"net/http"
	"os"

	"byfeature/customers"
	"byfeature/notifications"
	"byfeature/orders"
	"byfeature/products"
)

func main() {
	productStore := products.NewStore()
	customerStore := customers.NewStore()
	notificationStore := notifications.NewStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	products.Register(mux, productStore)
	customers.Register(mux, customerStore)
	notifications.Register(mux, notificationStore)
	orders.Register(mux, orders.NewService(productStore, customerStore, notificationStore))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("03-package-by-feature on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
