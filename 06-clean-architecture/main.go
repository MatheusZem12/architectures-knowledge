// main belongs to the outermost ring: it is the only code that knows all the others.
package main

import (
	"log"
	"net/http"
	"os"

	"cleanarch/adapters/controllers"
	"cleanarch/adapters/gateways"
	"cleanarch/frameworks/web"
	"cleanarch/usecases"
)

func main() {
	db := gateways.NewMemoryDB()
	payments := gateways.NewFakePayments()

	controller := controllers.New(controllers.UseCases{
		CreateProduct:     usecases.NewCreateProduct(db),
		GetProduct:        usecases.NewGetProduct(db),
		CreateCustomer:    usecases.NewCreateCustomer(db, db, db),
		GetCustomer:       usecases.NewGetCustomer(db),
		ListNotifications: usecases.NewListNotifications(db),
		PlaceOrder:        usecases.NewPlaceOrder(db, db, db, db, db, db),
		GetOrder:          usecases.NewGetOrder(db),
		PayOrder:          usecases.NewPayOrder(db, db, payments, db),
		CancelOrder:       usecases.NewCancelOrder(db, db, db, db),
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("06-clean-architecture on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, web.NewServer(controller)))
}
