package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status and sender-type constants.
const (
	InquiryStatusOpen   = "open"
	InquiryStatusClosed = "closed"

	SenderTypeBuyer  = "buyer"
	SenderTypeSeller = "seller"
)

// Inquiry represents a buyer→seller support thread scoped to one SKU the
// buyer has already purchased.
type Inquiry struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	BuyerAuth0ID  string    `json:"buyer_auth0_id"`
	SellerID      uuid.UUID `json:"seller_id"`
	SKUID         uuid.UUID `json:"sku_id"`
	ProductName   string    `json:"product_name"`
	SKUCode       string    `json:"sku_code"`
	Subject       string    `json:"subject"`
	Status        string    `json:"status"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// UnreadCount is populated on list queries; it counts messages from the
	// *other* party that the current viewer has not yet marked as read.
	UnreadCount int `json:"unread_count,omitempty"`
}

// InquiryMessage is one message within a thread.
type InquiryMessage struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	InquiryID  uuid.UUID  `json:"inquiry_id"`
	SenderType string     `json:"sender_type"`
	SenderID   string     `json:"sender_id"`
	Body       string     `json:"body"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// InquiryWithMessages bundles an inquiry with its full message history.
type InquiryWithMessages struct {
	Inquiry
	Messages []InquiryMessage `json:"messages"`
}

// CreateInquiryInput is what the service layer consumes when a buyer opens
// a new thread. ProductName/SKUCode are filled by the service layer from
// the purchase-check response so the caller cannot forge them.
type CreateInquiryInput struct {
	SellerID    uuid.UUID `json:"seller_id"`
	SKUID       uuid.UUID `json:"sku_id"`
	Subject     string    `json:"subject"`
	InitialBody string    `json:"initial_body"`
}

// PostMessageInput carries a reply body into the service layer. SenderType /
// SenderID are set by the handler based on the tenant context, not the wire
// payload, so buyers cannot impersonate sellers and vice versa.
type PostMessageInput struct {
	InquiryID  uuid.UUID
	SenderType string
	SenderID   string
	Body       string
}
