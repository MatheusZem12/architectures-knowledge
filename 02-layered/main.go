// Composition: assembles the layers from the bottom up and starts the server.
package main

import (
	"log"
	"net/http"
	"os"

	"layered/internal/handler"
	"layered/internal/integration"
	"layered/internal/repository"
	"layered/internal/service"
)

func main() {
	products := repository.NewProductRepository()
	customers := repository.NewCustomerRepository()
	orders := repository.NewOrderRepository()
	notifications := repository.NewNotificationRepository()

	router := handler.NewRouter(
		service.NewProductService(products),
		service.NewCustomerService(customers),
		service.NewOrderService(orders, products, customers, notifications, integration.NewPaymentGateway()),
		service.NewNotificationService(notifications),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("02-layered on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
