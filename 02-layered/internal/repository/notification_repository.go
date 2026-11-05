package repository

import (
	"sync"

	"layered/internal/model"
)

type NotificationRepository struct {
	mu    sync.RWMutex
	items map[string][]model.Notification
}

func NewNotificationRepository() *NotificationRepository {
	return &NotificationRepository{items: map[string][]model.Notification{}}
}

func (r *NotificationRepository) Create(n model.Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[n.CustomerID] = append(r.items[n.CustomerID], n)
}

func (r *NotificationRepository) FindByCustomer(customerID string) []model.Notification {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]model.Notification{}, r.items[customerID]...)
}
