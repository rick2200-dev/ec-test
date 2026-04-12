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
	Create(ctx context.Context, tenantID uuid.UUID, inq *domain.Inquiry, msg *domain.InquiryMessage) (*domain.InquiryWithMessages, error)
	GetByID(ctx context.Context, tenantID, inquiryID uuid.UUID) (*domain.InquiryWithMessages, error)
	ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error)
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error)
	AppendMessage(ctx context.Context, tenantID uuid.UUID, msg *domain.InquiryMessage) error
	MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string) error
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
	CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error)
}
