package controllers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mvc/models"
)

// One controller and one model, with two views: the format is the view's decision alone.
func TestSameRouteRendersJSONOrHTML(t *testing.T) {
	db := models.NewDB(models.FakeGateway)
	product, _ := db.CreateProduct("<b>Keyboard</b>", 100, 1)
	server := httptest.NewServer(New(db).Routes())
	defer server.Close()

	get := func(accept string) string {
		req, _ := http.NewRequest("GET", server.URL+"/products/"+product.ID, nil)
		req.Header.Set("Accept", accept)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return string(body)
	}

	if got := get("application/json"); !strings.Contains(got, `"price_cents":100`) {
		t.Fatalf("JSON view: %s", got)
	}
	got := get("text/html")
	if !strings.Contains(got, "In stock: 1") {
		t.Fatalf("HTML view: %s", got)
	}
	if strings.Contains(got, "<b>Keyboard") {
		t.Fatalf("the HTML view must escape user data: %s", got)
	}
}
