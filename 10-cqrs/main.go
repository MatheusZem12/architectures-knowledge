// main connects the two sides: what the write side publishes goes to the read side.
package main

import (
	"log"
	"net/http"
	"os"

	"cqrs/api"
	"cqrs/read"
	"cqrs/write"
)

func main() {
	views := read.NewStore()
	commands := write.NewHandler(views, write.FakeGateway{})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("10-cqrs on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, api.New(commands, views)))
}
