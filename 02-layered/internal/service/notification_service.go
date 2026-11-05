package service

import (
	"layered/internal/model"
	"layered/internal/repository"
)

type NotificationService struct {
	notifications *repository.NotificationRepository
}

func NewNotificationService(notifications *repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

func (s *NotificationService) ListByCustomer(customerID string) []model.Notification {
	return s.notifications.FindByCustomer(customerID)
}
