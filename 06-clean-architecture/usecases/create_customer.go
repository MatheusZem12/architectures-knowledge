package usecases

import "cleanarch/entities"

type CreateCustomerInput struct {
	Name  string
	Email string
}

type CreateCustomer struct {
	customers CustomerWriter
	emails    EmailChecker
	tx        Transactor
}

func NewCreateCustomer(customers CustomerWriter, emails EmailChecker, tx Transactor) *CreateCustomer {
	return &CreateCustomer{customers: customers, emails: emails, tx: tx}
}

func (uc *CreateCustomer) Execute(in CreateCustomerInput) (CustomerOutput, error) {
	customer, err := entities.NewCustomer(in.Name, in.Email)
	if err != nil {
		return CustomerOutput{}, err
	}
	var out CustomerOutput
	err = uc.tx.Atomic(func() error {
		// R03
		if uc.emails.EmailExists(in.Email) {
			return entities.Fail(entities.CodeEmailTaken, "email already registered")
		}
		out = customerOutput(uc.customers.AddCustomer(customer))
		return nil
	})
	return out, err
}
