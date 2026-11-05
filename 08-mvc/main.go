// main creates the model and hands it to the controllers.
package main

import (
	"log"
	"net/http"
	"os"

	"mvc/controllers"
	"mvc/models"
)

func main() {
	db := models.NewDB(models.FakeGateway)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("08-mvc on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, controllers.New(db).Routes()))
}
