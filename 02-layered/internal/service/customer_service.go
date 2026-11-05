package service

import (
	"strings"
	"sync"

	"layered/internal/model"
	"layered/internal/repository"
)

type CustomerService struct {
	mu        sync.Mutex // keeps the "check then create" of R03 free of races
	customers *repository.CustomerRepository
}

func NewCustomerService(customers *repository.CustomerRepository) *CustomerService {
	return &CustomerService{customers: customers}
}

func (s *CustomerService) Create(c model.Customer) (model.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// R02
	at := strings.Index(c.Email, "@")
	if strings.TrimSpace(c.Name) == "" || at <= 0 || at == len(c.Email)-1 {
		return model.Customer{}, fail("validation", "customer needs a name and a valid email")
	}
	// R03
	if s.customers.ExistsByEmail(c.Email) {
		return model.Customer{}, fail("email_taken", "email already registered")
	}
	return s.customers.Create(c), nil
}

func (s *CustomerService) Get(id string) (model.Customer, error) {
	c, ok := s.customers.FindByID(id)
	if !ok {
		return model.Customer{}, fail("not_found", "customer not found")
	}
	return c, nil
}
