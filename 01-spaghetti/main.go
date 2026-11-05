// Everything here is deliberately bad, yet it works: this file passes 100% of the contract
// suite. That is the point. Architecture does not show in today's behavior, but
// in the cost of changing tomorrow.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var mu sync.Mutex
var products = map[string]map[string]any{}
var customers = map[string]map[string]any{}
var orders = map[string]map[string]any{}
var notifs = map[string][]map[string]any{}
var counter = 0

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", handle)
	log.Println("01-spaghetti on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if parts[0] == "health" {
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	if parts[0] == "products" {
		if r.Method == "POST" && len(parts) == 1 {
			var d map[string]any
			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				w.WriteHeader(400)
				w.Write([]byte(`{"code":"invalid_json","error":"invalid JSON"}`))
				return
			}
			n, _ := d["name"].(string)
			p, _ := d["price_cents"].(float64)
			s, _ := d["stock"].(float64)
			// R01
			if strings.TrimSpace(n) == "" || p <= 0 || s < 0 {
				w.WriteHeader(422)
				w.Write([]byte(`{"code":"validation","error":"invalid product"}`))
				return
			}
			counter++
			id := fmt.Sprintf("%d", counter)
			products[id] = map[string]any{"id": id, "name": n, "price_cents": p, "stock": s}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(products[id])
			return
		}
		if r.Method == "GET" && len(parts) == 2 {
			if x, ok := products[parts[1]]; ok {
				json.NewEncoder(w).Encode(x)
			} else {
				w.WriteHeader(404)
				w.Write([]byte(`{"code":"not_found","error":"not found"}`))
			}
			return
		}
	}

	if parts[0] == "customers" {
		if r.Method == "POST" && len(parts) == 1 {
			var d map[string]any
			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				w.WriteHeader(400)
				w.Write([]byte(`{"code":"invalid_json","error":"invalid JSON"}`))
				return
			}
			n, _ := d["name"].(string)
			e, _ := d["email"].(string)
			at := strings.Index(e, "@")
			// R02
			if strings.TrimSpace(n) == "" || at <= 0 || at == len(e)-1 {
				w.WriteHeader(422)
				w.Write([]byte(`{"code":"validation","error":"invalid customer"}`))
				return
			}
			// R03
			for _, c := range customers {
				if strings.EqualFold(c["email"].(string), e) {
					w.WriteHeader(409)
					w.Write([]byte(`{"code":"email_taken","error":"email in use"}`))
					return
				}
			}
			counter++
			id := fmt.Sprintf("%d", counter)
			customers[id] = map[string]any{"id": id, "name": n, "email": e}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(customers[id])
			return
		}
		if r.Method == "GET" && len(parts) == 2 {
			if x, ok := customers[parts[1]]; ok {
				json.NewEncoder(w).Encode(x)
			} else {
				w.WriteHeader(404)
				w.Write([]byte(`{"code":"not_found","error":"not found"}`))
			}
			return
		}
		if r.Method == "GET" && len(parts) == 3 && parts[2] == "notifications" {
			if notifs[parts[1]] == nil {
				w.Write([]byte("[]"))
				return
			}
			json.NewEncoder(w).Encode(notifs[parts[1]])
			return
		}
	}

	if parts[0] == "orders" {
		if r.Method == "POST" && len(parts) == 1 {
			var d map[string]any
			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				w.WriteHeader(400)
				w.Write([]byte(`{"code":"invalid_json","error":"invalid JSON"}`))
				return
			}
			its, _ := d["items"].([]any)
			// R04
			if len(its) == 0 {
				w.WriteHeader(422)
				w.Write([]byte(`{"code":"validation","error":"order without items"}`))
				return
			}
			seen := map[string]bool{}
			for _, i := range its {
				m, _ := i.(map[string]any)
				q, _ := m["quantity"].(float64)
				pid, _ := m["product_id"].(string)
				if q < 1 || q > 10 || seen[pid] {
					w.WriteHeader(422)
					w.Write([]byte(`{"code":"validation","error":"invalid item"}`))
					return
				}
				seen[pid] = true
			}
			// R05
			cid, _ := d["customer_id"].(string)
			if _, ok := customers[cid]; !ok {
				w.WriteHeader(422)
				w.Write([]byte(`{"code":"customer_not_found","error":"customer does not exist"}`))
				return
			}
			var st float64
			out := []map[string]any{}
			for _, i := range its {
				m := i.(map[string]any)
				pid, _ := m["product_id"].(string)
				pr, ok := products[pid]
				if !ok {
					w.WriteHeader(422)
					w.Write([]byte(`{"code":"product_not_found","error":"product does not exist"}`))
					return
				}
				// R06
				if pr["stock"].(float64) < m["quantity"].(float64) {
					w.WriteHeader(409)
					w.Write([]byte(`{"code":"insufficient_stock","error":"out of stock"}`))
					return
				}
				st += pr["price_cents"].(float64) * m["quantity"].(float64)
				// R07
				out = append(out, map[string]any{"product_id": pid, "quantity": m["quantity"], "unit_price_cents": pr["price_cents"]})
			}
			// R08
			var disc float64
			cp, _ := d["coupon"].(string)
			if cp == "WELCOME10" {
				for _, o := range orders {
					if o["customer_id"] == cid && o["status"] != "cancelled" {
						w.WriteHeader(422)
						w.Write([]byte(`{"code":"coupon_not_eligible","error":"only on the first order"}`))
						return
					}
				}
				disc = float64(int(st) * 10 / 100)
			} else if cp == "BIG20" {
				if st < 50000 {
					w.WriteHeader(422)
					w.Write([]byte(`{"code":"coupon_not_eligible","error":"subtotal too low"}`))
					return
				}
				disc = float64(int(st) * 20 / 100)
			} else if cp != "" {
				w.WriteHeader(422)
				w.Write([]byte(`{"code":"invalid_coupon","error":"invalid coupon"}`))
				return
			}
			// R09
			var ship float64 = 2000
			if st-disc >= 20000 {
				ship = 0
			}
			// R06: only deduct after everything is validated
			for _, i := range its {
				m := i.(map[string]any)
				pr := products[m["product_id"].(string)]
				pr["stock"] = pr["stock"].(float64) - m["quantity"].(float64)
			}
			counter++
			id := fmt.Sprintf("%d", counter)
			orders[id] = map[string]any{
				"id": id, "customer_id": cid, "status": "pending", "items": out, "coupon": cp,
				// R10
				"subtotal_cents": st, "discount_cents": disc, "shipping_cents": ship, "total_cents": st - disc + ship,
			}
			// R14
			notifs[cid] = append(notifs[cid], map[string]any{"customer_id": cid, "order_id": id, "type": "order_created"})
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(orders[id])
			return
		}
		if len(parts) >= 2 {
			o, ok := orders[parts[1]]
			if !ok {
				w.WriteHeader(404)
				w.Write([]byte(`{"code":"not_found","error":"not found"}`))
				return
			}
			if r.Method == "GET" && len(parts) == 2 {
				json.NewEncoder(w).Encode(o)
				return
			}
			if r.Method == "POST" && len(parts) == 3 && parts[2] == "pay" {
				// R11
				if o["status"] != "pending" {
					w.WriteHeader(409)
					w.Write([]byte(`{"code":"invalid_transition","error":"cannot pay"}`))
					return
				}
				// R12: the "payment gateway" is an if in the middle of the handler
				if o["total_cents"].(float64) > 1000000 {
					w.WriteHeader(402)
					w.Write([]byte(`{"code":"payment_declined","error":"payment declined"}`))
					return
				}
				o["status"] = "paid"
				// R14
				cid := o["customer_id"].(string)
				notifs[cid] = append(notifs[cid], map[string]any{"customer_id": cid, "order_id": o["id"], "type": "order_paid"})
				json.NewEncoder(w).Encode(o)
				return
			}
			if r.Method == "POST" && len(parts) == 3 && parts[2] == "cancel" {
				// R11
				if o["status"] == "cancelled" {
					w.WriteHeader(409)
					w.Write([]byte(`{"code":"invalid_transition","error":"already cancelled"}`))
					return
				}
				o["status"] = "cancelled"
				// R13
				for _, it := range o["items"].([]map[string]any) {
					pr := products[it["product_id"].(string)]
					pr["stock"] = pr["stock"].(float64) + it["quantity"].(float64)
				}
				// R14
				cid := o["customer_id"].(string)
				notifs[cid] = append(notifs[cid], map[string]any{"customer_id": cid, "order_id": o["id"], "type": "order_cancelled"})
				json.NewEncoder(w).Encode(o)
				return
			}
		}
	}

	w.WriteHeader(404)
	w.Write([]byte(`{"code":"not_found","error":"route not found"}`))
}
