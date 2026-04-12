package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
)

// InquiryUseCase is the driving port (inbound) for inquiry operations.
// Handlers depend on this interface; *service.InquiryService satisfies it.
type InquiryUseCase interface {
	// CreateInquiry opens a new inquiry from a buyer to a seller with an initial message.
	CreateInquiry(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, in domain.CreateInquiryInput) (*domain.InquiryWithMessages, error)
	// PostMessage appends a message to an existing inquiry.
	// The actor can be a buyer (actorSellerID nil) or a seller (actorSellerID non-nil).
	PostMessage(ctx context.Context, tenantID uuid.UUID, actorAuth0ID string, actorSellerID *uuid.UUID, in domain.PostMessageInput) (*domain.InquiryMessage, error)
	// GetInquiry retrieves an inquiry with all messages, verifying that the actor is a participant.
	GetInquiry(ctx context.Context, tenantID, inquiryID uuid.UUID, actorAuth0ID string, actorSellerID *uuid.UUID) (*domain.InquiryWithMessages, error)
	// ListForBuyer returns a paginated list of inquiries for the authenticated buyer.
	ListForBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error)
	// ListForSeller returns a paginated list of inquiries for the given seller, optionally filtered by status.
	ListForSeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error)
	// MarkRead records that the actor has read the inquiry's messages up to the current time.
	MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string, actorAuth0ID string, actorSellerID *uuid.UUID) error
	// CloseInquiry closes the inquiry; only the seller (actorSellerID non-nil) may perform this action.
	CloseInquiry(ctx context.Context, tenantID, inquiryID uuid.UUID, actorSellerID *uuid.UUID) error
}
