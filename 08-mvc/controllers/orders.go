package controllers

import (
	"net/http"

	"mvc/models"
	"mvc/views"
)

func (c *Controller) createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID string `json:"customer_id"`
		Coupon     string `json:"coupon"`
		Items      []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if !decode(w, r, &in) {
		return
	}
	lines := make([]models.Line, len(in.Items))
	for i, item := range in.Items {
		lines[i] = models.Line{ProductID: item.ProductID, Quantity: item.Quantity}
	}
	order, err := c.db.PlaceOrder(in.CustomerID, lines, in.Coupon)
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Order(w, r, 201, order)
}

func (c *Controller) showOrder(w http.ResponseWriter, r *http.Request) {
	c.respond(w, r, c.db.FindOrder)
}

func (c *Controller) payOrder(w http.ResponseWriter, r *http.Request) {
	c.respond(w, r, c.db.PayOrder)
}

func (c *Controller) cancelOrder(w http.ResponseWriter, r *http.Request) {
	c.respond(w, r, c.db.CancelOrder)
}

func (c *Controller) respond(w http.ResponseWriter, r *http.Request, action func(string) (models.Order, error)) {
	order, err := action(r.PathValue("id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Order(w, r, 200, order)
}
