package controllers

import (
	"net/http"

	"mvc/views"
)

func (c *Controller) createCustomer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	customer, err := c.db.CreateCustomer(in.Name, in.Email)
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Customer(w, r, 201, customer)
}

func (c *Controller) showCustomer(w http.ResponseWriter, r *http.Request) {
	customer, err := c.db.FindCustomer(r.PathValue("id"))
	if err != nil {
		fail(w, r, err)
		return
	}
	views.Customer(w, r, 200, customer)
}

func (c *Controller) listNotifications(w http.ResponseWriter, r *http.Request) {
	views.Notifications(w, r, c.db.NotificationsOf(r.PathValue("id")))
}
