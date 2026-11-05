package usecases

import "cleanarch/entities"

type GetProduct struct {
	products ProductReader
}

func NewGetProduct(products ProductReader) *GetProduct {
	return &GetProduct{products: products}
}

func (uc *GetProduct) Execute(id string) (ProductOutput, error) {
	product, ok := uc.products.FindProduct(id)
	if !ok {
		return ProductOutput{}, entities.Fail(entities.CodeNotFound, "product not found")
	}
	return productOutput(product), nil
}
