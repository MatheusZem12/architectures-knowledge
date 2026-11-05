// bench measures POST /orders latency, sequentially, against an implementation already running.
//
//	BASE_URL=http://localhost:8080 go run ./cmd/bench
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"
)

const (
	warmup   = 500
	measured = 3000
)

var client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 4}}

func post(url string, payload any, wantStatus int) map[string]any {
	body, _ := json.Marshal(payload)
	res, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		log.Fatalf("status %d on %s: %s", res.StatusCode, url, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func main() {
	base := os.Getenv("BASE_URL")
	product := post(base+"/products", map[string]any{"name": "Bench", "price_cents": 1000, "stock": 1_000_000}, 201)
	customer := post(base+"/customers", map[string]any{"name": "Bench", "email": fmt.Sprintf("bench%d@example.com", time.Now().UnixNano())}, 201)
	order := map[string]any{
		"customer_id": customer["id"],
		"items":       []map[string]any{{"product_id": product["id"], "quantity": 1}},
	}

	for range warmup {
		post(base+"/orders", order, 201)
	}
	samples := make([]time.Duration, measured)
	started := time.Now()
	for i := range samples {
		start := time.Now()
		post(base+"/orders", order, 201)
		samples[i] = time.Since(start)
	}
	elapsed := time.Since(started)
	slices.Sort(samples)

	_ = json.NewEncoder(os.Stdout).Encode(map[string]float64{
		"p50_us": float64(samples[measured/2].Microseconds()),
		"p99_us": float64(samples[measured*99/100].Microseconds()),
		"rps":    float64(measured) / elapsed.Seconds(),
	})
}
