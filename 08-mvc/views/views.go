// Package views is the V in MVC: it decides what the answer looks like, and nothing else.
//
// The same data can be shown in more than one way. Each view here answers JSON by default,
// and HTML when the client asks for it with "Accept: text/html".
package views

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"mvc/models"
)

//go:embed templates/*.html
var files embed.FS

var pages = template.Must(template.ParseFS(files, "templates/*.html"))

type productJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
	Stock      int    `json:"stock"`
}

type customerJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type itemJSON struct {
	ProductID      string `json:"product_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
}

type orderJSON struct {
	ID            string     `json:"id"`
	CustomerID    string     `json:"customer_id"`
	Status        string     `json:"status"`
	Items         []itemJSON `json:"items"`
	Coupon        string     `json:"coupon"`
	SubtotalCents int        `json:"subtotal_cents"`
	DiscountCents int        `json:"discount_cents"`
	ShippingCents int        `json:"shipping_cents"`
	TotalCents    int        `json:"total_cents"`
}

type notificationJSON struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	Type       string `json:"type"`
}

func Product(w http.ResponseWriter, r *http.Request, status int, p models.Product) {
	render(w, r, status, "product.html", p, productJSON(p))
}

func Customer(w http.ResponseWriter, r *http.Request, status int, c models.Customer) {
	render(w, r, status, "customer.html", c, customerJSON(c))
}

func Order(w http.ResponseWriter, r *http.Request, status int, o models.Order) {
	items := make([]itemJSON, len(o.Items))
	for i, item := range o.Items {
		items[i] = itemJSON(item)
	}
	body := orderJSON{o.ID, o.CustomerID, o.Status, items, o.Coupon, o.SubtotalCents, o.DiscountCents, o.ShippingCents, o.TotalCents}
	render(w, r, status, "order.html", o, body)
}

func Notifications(w http.ResponseWriter, r *http.Request, list []models.Notification) {
	body := make([]notificationJSON, len(list))
	for i, n := range list {
		body[i] = notificationJSON(n)
	}
	render(w, r, 200, "notifications.html", list, body)
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	body := map[string]string{"code": code, "error": message}
	render(w, r, status, "error.html", body, body)
}

func Health(w http.ResponseWriter) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func render(w http.ResponseWriter, r *http.Request, status int, page string, htmlData, jsonData any) {
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		writeJSON(w, status, jsonData)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pages.ExecuteTemplate(w, page, htmlData)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
