package kernel_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"microkernel/kernel"
)

// halfOff is a brand new feature, written here without touching the kernel.
type halfOff struct{}

func (halfOff) Name() string { return "half-off" }

func (halfOff) Install(registry *kernel.Registry) { registry.AddCoupon("HALF", halfOff{}) }

func (halfOff) Discount(subtotalCents int, _ bool) (int, error) { return subtotalCents / 2, nil }

func post(t *testing.T, url, body string) (int, string) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(raw)
}

func TestKernelWorksAloneAndPluginsOnlyAdd(t *testing.T) {
	core := kernel.New()
	server := httptest.NewServer(core.Handler())
	defer server.Close()
	post(t, server.URL+"/products", `{"name":"Keyboard","price_cents":1000,"stock":5}`)
	post(t, server.URL+"/customers", `{"name":"Ana","email":"ana@example.com"}`)
	order := `{"customer_id":"cust-2","coupon":"HALF","items":[{"product_id":"prod-1","quantity":1}]}`

	if status, body := post(t, server.URL+"/orders", order); status != 422 || !strings.Contains(body, "invalid_coupon") {
		t.Fatalf("without the plugin: %d %s", status, body)
	}

	core.Load(halfOff{})

	if status, body := post(t, server.URL+"/orders", order); status != 201 || !strings.Contains(body, `"discount_cents":500`) {
		t.Fatalf("with the plugin: %d %s", status, body)
	}
}
