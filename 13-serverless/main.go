// main is the deployment description, like a serverless.yml: which function answers which route.
// There is no application here, and no server of ours. Only a table handed to the "cloud".
//
// COLD_START_MS simulates the delay of starting a new instance. It defaults to zero.
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"serverless/cloud"
	"serverless/functions/cancelorder"
	"serverless/functions/createcustomer"
	"serverless/functions/createproduct"
	"serverless/functions/getcustomer"
	"serverless/functions/getorder"
	"serverless/functions/getproduct"
	"serverless/functions/listnotifications"
	"serverless/functions/payorder"
	"serverless/functions/placeorder"
	"serverless/functions/sendnotification"
)

func main() {
	coldStartMs, _ := strconv.Atoi(os.Getenv("COLD_START_MS"))
	provider := cloud.New(time.Duration(coldStartMs)*time.Millisecond, time.Minute)

	provider.Route("POST /products", "createproduct", createproduct.Handler)
	provider.Route("GET /products/{id}", "getproduct", getproduct.Handler)
	provider.Route("POST /customers", "createcustomer", createcustomer.Handler)
	provider.Route("GET /customers/{id}", "getcustomer", getcustomer.Handler)
	provider.Route("GET /customers/{id}/notifications", "listnotifications", listnotifications.Handler)
	provider.Route("POST /orders", "placeorder", placeorder.Handler)
	provider.Route("GET /orders/{id}", "getorder", getorder.Handler)
	provider.Route("POST /orders/{id}/pay", "payorder", payorder.Handler)
	provider.Route("POST /orders/{id}/cancel", "cancelorder", cancelorder.Handler)
	provider.Trigger("order-events", "sendnotification", sendnotification.Handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("13-serverless on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, provider.Gateway()))
}
