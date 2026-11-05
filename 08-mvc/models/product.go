package models

import "strings"

type Product struct {
	ID         string
	Name       string
	PriceCents int
	Stock      int
}

func (db *DB) CreateProduct(name string, priceCents, stock int) (Product, error) {
	// R01
	if strings.TrimSpace(name) == "" || priceCents <= 0 || stock < 0 {
		return Product{}, Fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	product := Product{ID: db.nextID("prod"), Name: name, PriceCents: priceCents, Stock: stock}
	db.products[product.ID] = product
	return product, nil
}

func (db *DB) FindProduct(id string) (Product, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	product, ok := db.products[id]
	if !ok {
		return Product{}, Fail("not_found", "product not found")
	}
	return product, nil
}
