package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
)

// InquiryUseCase is the driving port (inbound) for inquiry operations.
// Handlers depend on this interface; *service.InquiryService satisfies it.
type InquiryUseCase interface {
	CreateInquiry(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, in domain.CreateInquiryInput) (*domain.InquiryWithMessages, error)
	PostMessage(ctx context.Context, tenantID uuid.UUID, actorAuth0ID string, actorSellerID *uuid.UUID, in domain.PostMessageInput) (*domain.InquiryMessage, error)
	GetInquiry(ctx context.Context, tenantID, inquiryID uuid.UUID, actorAuth0ID string, actorSellerID *uuid.UUID) (*domain.InquiryWithMessages, error)
	ListForBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error)
	ListForSeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error)
	MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string, actorAuth0ID string, actorSellerID *uuid.UUID) error
	CloseInquiry(ctx context.Context, tenantID, inquiryID uuid.UUID, actorSellerID *uuid.UUID) error
}
