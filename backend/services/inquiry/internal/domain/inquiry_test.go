package domain_test

import (
	"testing"

	"github.com/google/uuid"

	domain "github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
)

func TestInquiryStatusConstants(t *testing.T) {
	t.Parallel()

	if domain.InquiryStatusOpen != "open" {
		t.Errorf("InquiryStatusOpen = %q, want %q", domain.InquiryStatusOpen, "open")
	}
	if domain.InquiryStatusClosed != "closed" {
		t.Errorf("InquiryStatusClosed = %q, want %q", domain.InquiryStatusClosed, "closed")
	}
}

func TestSenderTypeConstants(t *testing.T) {
	t.Parallel()

	if domain.SenderTypeBuyer != "buyer" {
		t.Errorf("SenderTypeBuyer = %q, want %q", domain.SenderTypeBuyer, "buyer")
	}
	if domain.SenderTypeSeller != "seller" {
		t.Errorf("SenderTypeSeller = %q, want %q", domain.SenderTypeSeller, "seller")
	}
}

func TestInquiryWithMessages_EmptyMessages(t *testing.T) {
	t.Parallel()

	iwm := domain.InquiryWithMessages{
		Inquiry: domain.Inquiry{
			ID:     uuid.New(),
			Status: domain.InquiryStatusOpen,
		},
	}

	if iwm.Messages != nil {
		t.Errorf("Messages = %v, want nil for zero-value slice", iwm.Messages)
	}
	if len(iwm.Messages) != 0 {
		t.Errorf("len(Messages) = %d, want 0", len(iwm.Messages))
	}
}

func TestDomainErrors_NotNil(t *testing.T) {
	t.Parallel()

	errs := map[string]error{
		"ErrInquiryNotFound":  domain.ErrInquiryNotFound,
		"ErrInquiryClosed":    domain.ErrInquiryClosed,
		"ErrNotParticipant":   domain.ErrNotParticipant,
		"ErrPurchaseRequired": domain.ErrPurchaseRequired,
		"ErrInvalidSenderType": domain.ErrInvalidSenderType,
		"ErrInvalidReaderType": domain.ErrInvalidReaderType,
	}

	for name, err := range errs {
		if err == nil {
			t.Errorf("%s is nil, want non-nil error", name)
		}
	}
}

func TestDomainErrors_Distinct(t *testing.T) {
	t.Parallel()

	errs := []error{
		domain.ErrInquiryNotFound,
		domain.ErrInquiryClosed,
		domain.ErrNotParticipant,
		domain.ErrPurchaseRequired,
		domain.ErrInvalidSenderType,
		domain.ErrInvalidReaderType,
	}

	seen := make(map[string]bool, len(errs))
	for _, err := range errs {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}

func TestCreateInquiryInput_Fields(t *testing.T) {
	t.Parallel()

	sellerID := uuid.New()
	skuID := uuid.New()

	input := domain.CreateInquiryInput{
		SellerID:    sellerID,
		SKUID:       skuID,
		Subject:     "Shipping question",
		InitialBody: "When will my order arrive?",
	}

	if input.SellerID != sellerID {
		t.Errorf("SellerID = %v, want %v", input.SellerID, sellerID)
	}
	if input.SKUID != skuID {
		t.Errorf("SKUID = %v, want %v", input.SKUID, skuID)
	}
	if input.Subject != "Shipping question" {
		t.Errorf("Subject = %q, want %q", input.Subject, "Shipping question")
	}
	if input.InitialBody != "When will my order arrive?" {
		t.Errorf("InitialBody = %q, want %q", input.InitialBody, "When will my order arrive?")
	}
}

func TestPostMessageInput_Fields(t *testing.T) {
	t.Parallel()

	inquiryID := uuid.New()

	input := domain.PostMessageInput{
		InquiryID:  inquiryID,
		SenderType: domain.SenderTypeBuyer,
		SenderID:   "auth0|abc123",
		Body:       "Thank you for the reply.",
	}

	if input.InquiryID != inquiryID {
		t.Errorf("InquiryID = %v, want %v", input.InquiryID, inquiryID)
	}
	if input.SenderType != domain.SenderTypeBuyer {
		t.Errorf("SenderType = %q, want %q", input.SenderType, domain.SenderTypeBuyer)
	}
	if input.SenderID != "auth0|abc123" {
		t.Errorf("SenderID = %q, want %q", input.SenderID, "auth0|abc123")
	}
	if input.Body != "Thank you for the reply." {
		t.Errorf("Body = %q, want %q", input.Body, "Thank you for the reply.")
	}
}
