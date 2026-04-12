// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the inquiry service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
)

// InquiryStore is the driven port for inquiry persistence.
// *repository.InquiryRepository satisfies this interface.
type InquiryStore interface {
	// Create persists a new inquiry along with its first message in a single operation.
	Create(ctx context.Context, tenantID uuid.UUID, inq *domain.Inquiry, msg *domain.InquiryMessage) (*domain.InquiryWithMessages, error)
	// GetByID retrieves an inquiry and all its messages by inquiry ID within the tenant.
	GetByID(ctx context.Context, tenantID, inquiryID uuid.UUID) (*domain.InquiryWithMessages, error)
	// ListByBuyer returns a paginated list of inquiries belonging to the buyer, ordered by latest activity.
	ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error)
	// ListBySeller returns a paginated list of inquiries directed at the seller, optionally filtered by status.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error)
	// AppendMessage adds a new message to an existing inquiry.
	AppendMessage(ctx context.Context, tenantID uuid.UUID, msg *domain.InquiryMessage) error
	// MarkRead marks messages as read by the specified reader type ("buyer" or "seller").
	MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string) error
	// Close sets the inquiry status to closed.
	Close(ctx context.Context, tenantID, inquiryID uuid.UUID) error
}

// PurchaseCheckResult is the result of a purchase verification against the
// order service. Defined here so both the app layer and the httpclient adapter
// share the same type without a circular import.
type PurchaseCheckResult struct {
	Purchased       bool      `json:"purchased"`
	EarliestOrderID uuid.UUID `json:"earliest_order_id,omitempty"`
	ProductName     string    `json:"product_name,omitempty"`
	SKUCode         string    `json:"sku_code,omitempty"`
}

// PurchaseChecker verifies whether a buyer has purchased a given SKU from a
// seller. *httpclient.OrderClient satisfies this interface.
type PurchaseChecker interface {
	// CheckPurchase verifies whether the buyer has purchased the given SKU from the seller.
	CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error)
}
