package usecases

import "cleanarch/entities"

type CreateProductInput struct {
	Name       string
	PriceCents int
	Stock      int
}

type CreateProduct struct {
	products ProductWriter
}

func NewCreateProduct(products ProductWriter) *CreateProduct {
	return &CreateProduct{products: products}
}

func (uc *CreateProduct) Execute(in CreateProductInput) (ProductOutput, error) {
	product, err := entities.NewProduct(in.Name, in.PriceCents, in.Stock)
	if err != nil {
		return ProductOutput{}, err
	}
	return productOutput(uc.products.AddProduct(product)), nil
}
