// Package controllers translates a generic request into a use case Input.
//
// Notice there is no net/http here: the input is a simple Request and the output is a simple
// Response. So controllers run in tests without any server, and changing the web
// framework only affects the outer ring.
package controllers

import (
	"encoding/json"

	"cleanarch/adapters/presenters"
	"cleanarch/usecases"
)

type Request struct {
	PathID string
	Body   []byte
}

type Response struct {
	Status int
	Body   any
}

type UseCases struct {
	CreateProduct     *usecases.CreateProduct
	GetProduct        *usecases.GetProduct
	CreateCustomer    *usecases.CreateCustomer
	GetCustomer       *usecases.GetCustomer
	ListNotifications *usecases.ListNotifications
	PlaceOrder        *usecases.PlaceOrder
	GetOrder          *usecases.GetOrder
	PayOrder          *usecases.PayOrder
	CancelOrder       *usecases.CancelOrder
}

type Controller struct {
	uc UseCases
}

func New(uc UseCases) *Controller { return &Controller{uc: uc} }

func (c *Controller) CreateProduct(r Request) Response {
	var in presenters.Product
	if err := json.Unmarshal(r.Body, &in); err != nil {
		return invalidJSON()
	}
	out, err := c.uc.CreateProduct.Execute(usecases.CreateProductInput{Name: in.Name, PriceCents: in.PriceCents, Stock: in.Stock})
	return result(201, presenters.Product(out), err)
}

func (c *Controller) GetProduct(r Request) Response {
	out, err := c.uc.GetProduct.Execute(r.PathID)
	return result(200, presenters.Product(out), err)
}

func (c *Controller) CreateCustomer(r Request) Response {
	var in presenters.Customer
	if err := json.Unmarshal(r.Body, &in); err != nil {
		return invalidJSON()
	}
	out, err := c.uc.CreateCustomer.Execute(usecases.CreateCustomerInput{Name: in.Name, Email: in.Email})
	return result(201, presenters.Customer(out), err)
}

func (c *Controller) GetCustomer(r Request) Response {
	out, err := c.uc.GetCustomer.Execute(r.PathID)
	return result(200, presenters.Customer(out), err)
}

func (c *Controller) ListNotifications(r Request) Response {
	return Response{200, presenters.PresentNotifications(c.uc.ListNotifications.Execute(r.PathID))}
}

func (c *Controller) PlaceOrder(r Request) Response {
	var body struct {
		CustomerID string `json:"customer_id"`
		Coupon     string `json:"coupon"`
		Items      []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(r.Body, &body); err != nil {
		return invalidJSON()
	}
	in := usecases.PlaceOrderInput{CustomerID: body.CustomerID, Coupon: body.Coupon}
	for _, item := range body.Items {
		in.Items = append(in.Items, usecases.PlaceOrderItem{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	out, err := c.uc.PlaceOrder.Execute(in)
	return result(201, presenters.PresentOrder(out), err)
}

func (c *Controller) GetOrder(r Request) Response {
	out, err := c.uc.GetOrder.Execute(r.PathID)
	return result(200, presenters.PresentOrder(out), err)
}

func (c *Controller) PayOrder(r Request) Response {
	out, err := c.uc.PayOrder.Execute(r.PathID)
	return result(200, presenters.PresentOrder(out), err)
}

func (c *Controller) CancelOrder(r Request) Response {
	out, err := c.uc.CancelOrder.Execute(r.PathID)
	return result(200, presenters.PresentOrder(out), err)
}

func result(status int, view any, err error) Response {
	if err != nil {
		errorStatus, errorView := presenters.PresentError(err)
		return Response{errorStatus, errorView}
	}
	return Response{status, view}
}

func invalidJSON() Response {
	return Response{400, presenters.ErrorView{Code: "invalid_json", Error: "invalid JSON"}}
}
