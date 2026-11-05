// Package contract tests a store implementation from the outside, over HTTP only.
//
// This suite imports nothing from the architectures: that is what guarantees they all do
// the same thing, however different their code may be.
//
//	BASE_URL=http://localhost:8080 go test ./...
package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	baseURL = strings.TrimRight(os.Getenv("BASE_URL"), "/")
	seq     atomic.Int64
)

type obj = map[string]any

func TestMain(m *testing.M) {
	if baseURL == "" {
		fmt.Println("set BASE_URL, for example: BASE_URL=http://localhost:8080 go test ./...")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// ---------- helpers ----------

type response struct {
	status int
	body   obj
	list   []obj
	raw    string
}

func call(t *testing.T, method, path string, payload any) response {
	t.Helper()
	var body io.Reader
	switch p := payload.(type) {
	case nil:
	case string:
		body = strings.NewReader(p)
	default:
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	out := response{status: res.StatusCode, raw: string(data)}
	if strings.HasPrefix(strings.TrimSpace(out.raw), "[") {
		_ = json.Unmarshal(data, &out.list)
	} else {
		_ = json.Unmarshal(data, &out.body)
	}
	return out
}

func expect(t *testing.T, r response, status int, code string) {
	t.Helper()
	if r.status != status {
		t.Fatalf("status = %d, want %d; body: %s", r.status, status, r.raw)
	}
	if code != "" && r.body["code"] != code {
		t.Fatalf("code = %v, want %s; body: %s", r.body["code"], code, r.raw)
	}
}

func cents(t *testing.T, o obj, field string) int {
	t.Helper()
	v, ok := o[field].(float64)
	if !ok {
		t.Fatalf("field %s missing or not numeric in %v", field, o)
	}
	return int(v)
}

func id(t *testing.T, o obj) string {
	t.Helper()
	s, ok := o["id"].(string)
	if !ok || s == "" {
		t.Fatalf("id missing in %v", o)
	}
	return s
}

func newProduct(t *testing.T, price, stock int) string {
	t.Helper()
	r := call(t, "POST", "/products", obj{"name": "Product", "price_cents": price, "stock": stock})
	expect(t, r, 201, "")
	return id(t, r.body)
}

func newCustomer(t *testing.T) string {
	t.Helper()
	email := fmt.Sprintf("customer%d-%d@example.com", time.Now().UnixNano(), seq.Add(1))
	r := call(t, "POST", "/customers", obj{"name": "Customer", "email": email})
	expect(t, r, 201, "")
	return id(t, r.body)
}

func item(product string, quantity int) obj {
	return obj{"product_id": product, "quantity": quantity}
}

func placeOrder(t *testing.T, customer string, coupon string, items ...obj) response {
	t.Helper()
	payload := obj{"customer_id": customer, "items": items}
	if coupon != "" {
		payload["coupon"] = coupon
	}
	return call(t, "POST", "/orders", payload)
}

func stockOf(t *testing.T, product string) int {
	t.Helper()
	r := call(t, "GET", "/products/"+product, nil)
	expect(t, r, 200, "")
	return cents(t, r.body, "stock")
}

// waitNotifications waits up to 2 s for the expected types (R14 allows asynchronous delivery).
func waitNotifications(t *testing.T, customer string, want ...string) {
	t.Helper()
	var got []string
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		r := call(t, "GET", "/customers/"+customer+"/notifications", nil)
		expect(t, r, 200, "")
		got = got[:0]
		for _, n := range r.list {
			got = append(got, fmt.Sprint(n["type"]))
		}
		if sameSet(got, want) {
			return
		}
	}
	t.Fatalf("notifications = %v, want %v", got, want)
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := map[string]int{}
	for _, x := range a {
		count[x]++
	}
	for _, x := range b {
		count[x]--
		if count[x] < 0 {
			return false
		}
	}
	return true
}

// ---------- basic endpoints ----------

func TestHealth(t *testing.T) {
	expect(t, call(t, "GET", "/health", nil), 200, "")
}

func TestProductRoundTrip(t *testing.T) {
	created := call(t, "POST", "/products", obj{"name": "Keyboard", "price_cents": 15000, "stock": 10})
	expect(t, created, 201, "")
	got := call(t, "GET", "/products/"+id(t, created.body), nil)
	expect(t, got, 200, "")
	if got.body["name"] != "Keyboard" || cents(t, got.body, "price_cents") != 15000 || cents(t, got.body, "stock") != 10 {
		t.Fatalf("unexpected product: %s", got.raw)
	}
}

func TestCustomerRoundTrip(t *testing.T) {
	email := fmt.Sprintf("ida%d@example.com", time.Now().UnixNano())
	created := call(t, "POST", "/customers", obj{"name": "Ida", "email": email})
	expect(t, created, 201, "")
	got := call(t, "GET", "/customers/"+id(t, created.body), nil)
	expect(t, got, 200, "")
	if got.body["name"] != "Ida" || got.body["email"] != email {
		t.Fatalf("unexpected customer: %s", got.raw)
	}
}

func TestNotFound(t *testing.T) {
	for _, path := range []string{"/products/missing", "/customers/missing", "/orders/missing"} {
		expect(t, call(t, "GET", path, nil), 404, "not_found")
	}
	expect(t, call(t, "POST", "/orders/missing/pay", nil), 404, "not_found")
	expect(t, call(t, "POST", "/orders/missing/cancel", nil), 404, "not_found")
}

func TestMalformedJSON(t *testing.T) {
	for _, path := range []string{"/products", "/customers", "/orders"} {
		expect(t, call(t, "POST", path, "{quebrado"), 400, "invalid_json")
	}
}

// ---------- rules ----------

func TestR01_ProductValidation(t *testing.T) {
	cases := []obj{
		{"name": "", "price_cents": 100, "stock": 1},
		{"name": "   ", "price_cents": 100, "stock": 1},
		{"name": "X", "price_cents": 0, "stock": 1},
		{"name": "X", "price_cents": -5, "stock": 1},
		{"name": "X", "price_cents": 100, "stock": -1},
	}
	for _, c := range cases {
		expect(t, call(t, "POST", "/products", c), 422, "validation")
	}
	expect(t, call(t, "POST", "/products", obj{"name": "X", "price_cents": 1, "stock": 0}), 201, "")
}

func TestR02_CustomerValidation(t *testing.T) {
	cases := []obj{
		{"name": "", "email": "a@b.com"},
		{"name": "Ana", "email": ""},
		{"name": "Ana", "email": "no-at-sign"},
		{"name": "Ana", "email": "@no-user"},
		{"name": "Ana", "email": "no-domain@"},
	}
	for _, c := range cases {
		expect(t, call(t, "POST", "/customers", c), 422, "validation")
	}
}

func TestR03_EmailIsUniqueIgnoringCase(t *testing.T) {
	email := fmt.Sprintf("Unique%d@Example.com", time.Now().UnixNano())
	expect(t, call(t, "POST", "/customers", obj{"name": "A", "email": email}), 201, "")
	expect(t, call(t, "POST", "/customers", obj{"name": "B", "email": email}), 409, "email_taken")
	expect(t, call(t, "POST", "/customers", obj{"name": "C", "email": strings.ToLower(email)}), 409, "email_taken")
}

func TestR04_OrderItems(t *testing.T) {
	customer, product := newCustomer(t), newProduct(t, 1000, 100)
	expect(t, placeOrder(t, customer, ""), 422, "validation")
	expect(t, placeOrder(t, customer, "", item(product, 0)), 422, "validation")
	expect(t, placeOrder(t, customer, "", item(product, 11)), 422, "validation")
	expect(t, placeOrder(t, customer, "", item(product, 1), item(product, 2)), 422, "validation")
	expect(t, placeOrder(t, customer, "", item(product, 10)), 201, "")
	if got := stockOf(t, product); got != 90 {
		t.Fatalf("stock = %d, want 90: invalid orders must not reserve", got)
	}
}

func TestR05_ReferencesMustExist(t *testing.T) {
	customer, product := newCustomer(t), newProduct(t, 1000, 5)
	expect(t, placeOrder(t, "missing", "", item(product, 1)), 422, "customer_not_found")
	expect(t, placeOrder(t, customer, "", item("missing", 1)), 422, "product_not_found")
	if got := stockOf(t, product); got != 5 {
		t.Fatalf("stock = %d, want 5", got)
	}
}

func TestR06_StockIsAllOrNothing(t *testing.T) {
	customer := newCustomer(t)
	plenty, scarce := newProduct(t, 1000, 10), newProduct(t, 1000, 1)
	expect(t, placeOrder(t, customer, "", item(plenty, 3), item(scarce, 2)), 409, "insufficient_stock")
	if got := stockOf(t, plenty); got != 10 {
		t.Fatalf("stock = %d, want 10: the partial reservation must be undone", got)
	}
	expect(t, placeOrder(t, customer, "", item(plenty, 3), item(scarce, 1)), 201, "")
	if a, b := stockOf(t, plenty), stockOf(t, scarce); a != 7 || b != 0 {
		t.Fatalf("stocks = %d and %d, want 7 and 0", a, b)
	}
}

func TestR06_ConcurrentOrdersNeverOversell(t *testing.T) {
	product := newProduct(t, 1000, 5)
	customers := make([]string, 20)
	for i := range customers {
		customers[i] = newCustomer(t)
	}
	var created, refused atomic.Int64
	var wg sync.WaitGroup
	for _, customer := range customers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := placeOrder(t, customer, "", item(product, 1))
			switch r.status {
			case 201:
				created.Add(1)
			case 409:
				refused.Add(1)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 5 || refused.Load() != 15 {
		t.Fatalf("created = %d, refused = %d; want 5 and 15", created.Load(), refused.Load())
	}
	if got := stockOf(t, product); got != 0 {
		t.Fatalf("stock = %d, want 0", got)
	}
}

func TestR07_R10_Totals(t *testing.T) {
	product := newProduct(t, 15000, 10)
	r := placeOrder(t, newCustomer(t), "", item(product, 2))
	expect(t, r, 201, "")
	if r.body["status"] != "pending" {
		t.Fatalf("status = %v, want pending", r.body["status"])
	}
	items, _ := r.body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("itens inesperados: %s", r.raw)
	}
	first := items[0].(map[string]any)
	if first["product_id"] != product || cents(t, first, "quantity") != 2 || cents(t, first, "unit_price_cents") != 15000 {
		t.Fatalf("item inesperado: %s", r.raw)
	}
	checkTotals(t, r.body, 30000, 0, 0, 30000)

	saved := call(t, "GET", "/orders/"+id(t, r.body), nil)
	expect(t, saved, 200, "")
	checkTotals(t, saved.body, 30000, 0, 0, 30000)
}

func checkTotals(t *testing.T, order obj, subtotal, discount, shipping, total int) {
	t.Helper()
	got := [4]int{cents(t, order, "subtotal_cents"), cents(t, order, "discount_cents"), cents(t, order, "shipping_cents"), cents(t, order, "total_cents")}
	want := [4]int{subtotal, discount, shipping, total}
	if got != want {
		t.Fatalf("subtotal/discount/shipping/total = %v, want %v", got, want)
	}
}

func TestR08_WelcomeCouponOnlyOnFirstOrder(t *testing.T) {
	product, customer := newProduct(t, 30000, 10), newCustomer(t)
	first := placeOrder(t, customer, "WELCOME10", item(product, 1))
	expect(t, first, 201, "")
	if first.body["coupon"] != "WELCOME10" {
		t.Fatalf("coupon was not saved: %s", first.raw)
	}
	checkTotals(t, first.body, 30000, 3000, 0, 27000)
	expect(t, placeOrder(t, customer, "WELCOME10", item(product, 1)), 422, "coupon_not_eligible")
}

func TestR08_CancelledOrdersDoNotCountAsFirst(t *testing.T) {
	product, customer := newProduct(t, 30000, 10), newCustomer(t)
	order := placeOrder(t, customer, "", item(product, 1))
	expect(t, order, 201, "")
	expect(t, call(t, "POST", "/orders/"+id(t, order.body)+"/cancel", nil), 200, "")
	expect(t, placeOrder(t, customer, "WELCOME10", item(product, 1)), 201, "")
}

func TestR08_BigOrderCoupon(t *testing.T) {
	product := newProduct(t, 50000, 10)
	r := placeOrder(t, newCustomer(t), "BIG20", item(product, 1))
	expect(t, r, 201, "")
	checkTotals(t, r.body, 50000, 10000, 0, 40000)

	cheap := newProduct(t, 49999, 10)
	expect(t, placeOrder(t, newCustomer(t), "BIG20", item(cheap, 1)), 422, "coupon_not_eligible")
}

func TestR08_BadCouponDoesNotConsumeStock(t *testing.T) {
	product, customer := newProduct(t, 1000, 5), newCustomer(t)
	expect(t, placeOrder(t, customer, "NOSUCHCOUPON", item(product, 2)), 422, "invalid_coupon")
	expect(t, placeOrder(t, customer, "BIG20", item(product, 2)), 422, "coupon_not_eligible")
	if got := stockOf(t, product); got != 5 {
		t.Fatalf("stock = %d, want 5: a refused coupon does not consume stock", got)
	}
}

func TestR09_ShippingBoundary(t *testing.T) {
	below := placeOrder(t, newCustomer(t), "", item(newProduct(t, 19999, 1), 1))
	expect(t, below, 201, "")
	checkTotals(t, below.body, 19999, 0, 2000, 21999)

	exact := placeOrder(t, newCustomer(t), "", item(newProduct(t, 20000, 1), 1))
	expect(t, exact, 201, "")
	checkTotals(t, exact.body, 20000, 0, 0, 20000)
}

func TestR09_ShippingUsesValueAfterDiscount(t *testing.T) {
	// 21000 would clear the threshold, but 21000 - 10% = 18900 does not: shipping is charged.
	r := placeOrder(t, newCustomer(t), "WELCOME10", item(newProduct(t, 21000, 1), 1))
	expect(t, r, 201, "")
	checkTotals(t, r.body, 21000, 2100, 2000, 20900)
}

func TestR10_PercentRoundsDown(t *testing.T) {
	// 10% of 999 = 99.9 cents, which becomes 99.
	r := placeOrder(t, newCustomer(t), "WELCOME10", item(newProduct(t, 999, 1), 1))
	expect(t, r, 201, "")
	checkTotals(t, r.body, 999, 99, 2000, 2900)
}

func TestR11_StateMachine(t *testing.T) {
	product := newProduct(t, 1000, 10)
	order := func() string {
		r := placeOrder(t, newCustomer(t), "", item(product, 1))
		expect(t, r, 201, "")
		return id(t, r.body)
	}

	paid := order()
	r := call(t, "POST", "/orders/"+paid+"/pay", nil)
	expect(t, r, 200, "")
	if r.body["status"] != "paid" {
		t.Fatalf("status = %v", r.body["status"])
	}
	expect(t, call(t, "POST", "/orders/"+paid+"/pay", nil), 409, "invalid_transition")

	r = call(t, "POST", "/orders/"+paid+"/cancel", nil)
	expect(t, r, 200, "")
	if r.body["status"] != "cancelled" {
		t.Fatalf("status = %v", r.body["status"])
	}
	expect(t, call(t, "POST", "/orders/"+paid+"/cancel", nil), 409, "invalid_transition")
	expect(t, call(t, "POST", "/orders/"+paid+"/pay", nil), 409, "invalid_transition")

	pending := order()
	expect(t, call(t, "POST", "/orders/"+pending+"/cancel", nil), 200, "")
	got := call(t, "GET", "/orders/"+pending, nil)
	if got.body["status"] != "cancelled" {
		t.Fatalf("status = %v", got.body["status"])
	}
}

func TestR12_PaymentDeclined(t *testing.T) {
	customer := newCustomer(t)
	// A total exactly at the limit passes.
	limit := placeOrder(t, newCustomer(t), "", item(newProduct(t, 1000000, 1), 1))
	expect(t, limit, 201, "")
	expect(t, call(t, "POST", "/orders/"+id(t, limit.body)+"/pay", nil), 200, "")

	order := placeOrder(t, customer, "", item(newProduct(t, 1000001, 1), 1))
	expect(t, order, 201, "")
	orderID := id(t, order.body)
	expect(t, call(t, "POST", "/orders/"+orderID+"/pay", nil), 402, "payment_declined")
	got := call(t, "GET", "/orders/"+orderID, nil)
	if got.body["status"] != "pending" {
		t.Fatalf("status = %v, want pending", got.body["status"])
	}
	time.Sleep(100 * time.Millisecond)
	waitNotifications(t, customer, "order_created")
}

func TestR13_CancelReturnsStock(t *testing.T) {
	product := newProduct(t, 1000, 10)
	pending := placeOrder(t, newCustomer(t), "", item(product, 4))
	expect(t, pending, 201, "")
	paid := placeOrder(t, newCustomer(t), "", item(product, 3))
	expect(t, paid, 201, "")
	expect(t, call(t, "POST", "/orders/"+id(t, paid.body)+"/pay", nil), 200, "")
	if got := stockOf(t, product); got != 3 {
		t.Fatalf("stock = %d, want 3", got)
	}
	expect(t, call(t, "POST", "/orders/"+id(t, pending.body)+"/cancel", nil), 200, "")
	expect(t, call(t, "POST", "/orders/"+id(t, paid.body)+"/cancel", nil), 200, "")
	if got := stockOf(t, product); got != 10 {
		t.Fatalf("stock = %d, want 10", got)
	}
	// A refused transition must not return the stock again.
	expect(t, call(t, "POST", "/orders/"+id(t, paid.body)+"/cancel", nil), 409, "invalid_transition")
	if got := stockOf(t, product); got != 10 {
		t.Fatalf("stock = %d, want 10 after the refused cancellation", got)
	}
}

func TestR14_Notifications(t *testing.T) {
	customer := newCustomer(t)
	r := call(t, "GET", "/customers/"+customer+"/notifications", nil)
	expect(t, r, 200, "")
	if strings.TrimSpace(r.raw) != "[]" {
		t.Fatalf("with no notifications the body must be [], got %q", r.raw)
	}
	order := placeOrder(t, customer, "", item(newProduct(t, 1000, 5), 1))
	expect(t, order, 201, "")
	waitNotifications(t, customer, "order_created")
	expect(t, call(t, "POST", "/orders/"+id(t, order.body)+"/pay", nil), 200, "")
	waitNotifications(t, customer, "order_created", "order_paid")
	expect(t, call(t, "POST", "/orders/"+id(t, order.body)+"/cancel", nil), 200, "")
	waitNotifications(t, customer, "order_created", "order_paid", "order_cancelled")
}
