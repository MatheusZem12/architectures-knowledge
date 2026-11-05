package models

type Notification struct {
	CustomerID string
	OrderID    string
	Type       string
}

func (db *DB) NotificationsOf(customerID string) []Notification {
	db.mu.Lock()
	defer db.mu.Unlock()
	return append([]Notification{}, db.notifications[customerID]...)
}

// notify must be called with the lock held.
func (db *DB) notify(order Order, kind string) {
	// R14
	db.notifications[order.CustomerID] = append(db.notifications[order.CustomerID], Notification{order.CustomerID, order.ID, kind})
}
