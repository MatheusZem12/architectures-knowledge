// main opens the event log and rebuilds everything from it. Set EVENT_FILE to keep the log on disk:
// after a restart, products, customers and orders all come back from the events alone.
package main

import (
	"log"
	"net/http"
	"os"

	"eventsourcing/api"
	"eventsourcing/commands"
	"eventsourcing/eventstore"
	"eventsourcing/projections"
)

func main() {
	store, err := eventstore.Open(os.Getenv("EVENT_FILE"))
	if err != nil {
		log.Fatal(err)
	}
	views := projections.New(store)
	handler := commands.New(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("12-event-sourcing on port %s, %d events replayed", port, store.Len())
	log.Fatal(http.ListenAndServe(":"+port, api.New(handler, views, store)))
}
