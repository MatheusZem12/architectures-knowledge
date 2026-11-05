// main lists the system's features, one per line. Adding a feature means adding a line here.
package main

import (
	"log"
	"net/http"
	"os"

	"verticalslice/features/cancelorder"
	"verticalslice/features/createcustomer"
	"verticalslice/features/createproduct"
	"verticalslice/features/getcustomer"
	"verticalslice/features/getorder"
	"verticalslice/features/getproduct"
	"verticalslice/features/listnotifications"
	"verticalslice/features/payorder"
	"verticalslice/features/placeorder"
	"verticalslice/store"
)

func main() {
	db := store.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })

	createproduct.Register(mux, db)
	getproduct.Register(mux, db)
	createcustomer.Register(mux, db)
	getcustomer.Register(mux, db)
	listnotifications.Register(mux, db)
	placeorder.Register(mux, db)
	getorder.Register(mux, db)
	payorder.Register(mux, db)
	cancelorder.Register(mux, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("11-vertical-slice on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
