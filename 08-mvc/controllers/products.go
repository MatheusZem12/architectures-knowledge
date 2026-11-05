package controllers

import (
	"net/http"

	"mvc/views"
)

func (c *Controller) createProduct(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		PriceCents int    `json:"price_cents"`
		Stock      int    `json:"stock"`
	}
	if !decode(w, r, &in) {
		return
	}
	product, err := c.db.CreateProduct(in.Name, in.PriceCents, in.Stock)
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Product(w, r, 201, product)
}

func (c *Controller) showProduct(w http.ResponseWriter, r *http.Request) {
	product, err := c.db.FindProduct(r.PathValue("id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Product(w, r, 200, product)
}
