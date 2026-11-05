package usecases

import "cleanarch/entities"

type GetOrder struct {
	orders OrderReader
}

func NewGetOrder(orders OrderReader) *GetOrder {
	return &GetOrder{orders: orders}
}

func (uc *GetOrder) Execute(id string) (OrderOutput, error) {
	order, ok := uc.orders.FindOrder(id)
	if !ok {
		return OrderOutput{}, entities.Fail(entities.CodeNotFound, "order not found")
	}
	return orderOutput(order), nil
}
