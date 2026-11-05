package usecases

type ListNotifications struct {
	notifications NotificationReader
}

func NewListNotifications(notifications NotificationReader) *ListNotifications {
	return &ListNotifications{notifications: notifications}
}

func (uc *ListNotifications) Execute(customerID string) []NotificationOutput {
	out := []NotificationOutput{}
	for _, n := range uc.notifications.NotificationsOf(customerID) {
		out = append(out, NotificationOutput(n))
	}
	return out
}
