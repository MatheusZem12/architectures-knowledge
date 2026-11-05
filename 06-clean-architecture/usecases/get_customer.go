package usecases

import "cleanarch/entities"

type GetCustomer struct {
	customers CustomerReader
}

func NewGetCustomer(customers CustomerReader) *GetCustomer {
	return &GetCustomer{customers: customers}
}

func (uc *GetCustomer) Execute(id string) (CustomerOutput, error) {
	customer, ok := uc.customers.FindCustomer(id)
	if !ok {
		return CustomerOutput{}, entities.Fail(entities.CodeNotFound, "customer not found")
	}
	return customerOutput(customer), nil
}
