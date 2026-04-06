package domain

import "time"

// NotificationType represents the type of notification.
type NotificationType string

const (
	OrderCreated   NotificationType = "order.created"
	OrderPaid      NotificationType = "order.paid"
	OrderShipped   NotificationType = "order.shipped"
	OrderDelivered NotificationType = "order.delivered"
	LowStock       NotificationType = "inventory.low_stock"
	SellerApproved NotificationType = "seller.approved"
)

// NotificationStatus represents the delivery status of a notification.
type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

// Notification represents a notification record.
type Notification struct {
	ID        string             `json:"id"`
	TenantID  string             `json:"tenant_id"`
	Type      NotificationType   `json:"type"`
	Recipient string             `json:"recipient"`
	Subject   string             `json:"subject"`
	Body      string             `json:"body"`
	Status    NotificationStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
}

// Template represents an email template configuration.
type Template struct {
	Name    string
	Subject string
	Body    string
}
