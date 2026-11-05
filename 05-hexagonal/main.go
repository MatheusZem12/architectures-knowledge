// main picks the adapters and plugs them into the core. Only here is it decided which technology
// is used. Run with STORE=json to swap persistence without touching the core.
package main

import (
	"log"
	"net/http"
	"os"

	"hexagonal/internal/adapters/in/httpapi"
	"hexagonal/internal/adapters/out/fakepayment"
	"hexagonal/internal/adapters/out/jsonfile"
	"hexagonal/internal/adapters/out/memory"
	"hexagonal/internal/core/app"
	"hexagonal/internal/core/ports"
)

func main() {
	var store interface {
		ports.ProductRepository
		ports.CustomerRepository
		ports.OrderRepository
		ports.NotificationRepository
	}
	if os.Getenv("STORE") == "json" {
		path := os.Getenv("DATA_FILE")
		if path == "" {
			path = "data.json"
		}
		fileStore, err := jsonfile.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		store = fileStore
	} else {
		store = memory.New()
	}

	shop := app.NewShop(store, store, store, store, fakepayment.New())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("05-hexagonal on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, httpapi.New(shop)))
}
