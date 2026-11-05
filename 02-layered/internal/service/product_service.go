package service

import (
	"strings"

	"layered/internal/model"
	"layered/internal/repository"
)

type ProductService struct {
	products *repository.ProductRepository
}

func NewProductService(products *repository.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

func (s *ProductService) Create(p model.Product) (model.Product, error) {
	// R01
	if strings.TrimSpace(p.Name) == "" || p.PriceCents <= 0 || p.Stock < 0 {
		return model.Product{}, fail("validation", "product needs a name, a positive price and non-negative stock")
	}
	return s.products.Create(p), nil
}

func (s *ProductService) Get(id string) (model.Product, error) {
	p, ok := s.products.FindByID(id)
	if !ok {
		return model.Product{}, fail("not_found", "product not found")
	}
	return p, nil
}
