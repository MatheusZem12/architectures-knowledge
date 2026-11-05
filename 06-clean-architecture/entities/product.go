package entities

import "strings"

type Product struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

func NewProduct(name string, priceCents, stock int) (Product, error) {
	// R01
	if strings.TrimSpace(name) == "" || priceCents <= 0 || stock < 0 {
		return Product{}, Fail(CodeValidation, "product needs a name, a positive price and non-negative stock")
	}
	return Product{Name: name, PriceCents: priceCents, Stock: stock}, nil
}

// Reserve applies R06 to a product.
func (p *Product) Reserve(quantity int) error {
	if p.Stock < quantity {
		return Fail(CodeInsufficientStock, "insufficient stock")
	}
	p.Stock -= quantity
	return nil
}

// Release applies R13 to a product.
func (p *Product) Release(quantity int) { p.Stock += quantity }
