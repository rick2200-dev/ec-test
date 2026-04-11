package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/repository"
)

const inquiryEventTopic = "inquiry-events"

// InquiryService holds the business logic for buyer↔seller threads.
type InquiryService struct {
	repo        *repository.InquiryRepository
	orderClient *OrderClient
	publisher   pubsub.Publisher
}

// NewInquiryService constructs an InquiryService.
func NewInquiryService(
	repo *repository.InquiryRepository,
	orderClient *OrderClient,
	publisher pubsub.Publisher,
) *InquiryService {
	return &InquiryService{
		repo:        repo,
		orderClient: orderClient,
		publisher:   publisher,
	}
}

// CreateInquiry opens a new thread (or returns the existing one) after
// verifying the buyer has a qualifying purchase for the SKU.
//
// The product_name / sku_code captured on the inquiry are snapshotted from
// the order line the buyer already paid for, so catalog edits later on
// cannot rewrite history on the thread.
func (s *InquiryService) CreateInquiry(
	ctx context.Context,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	in domain.CreateInquiryInput,
) (*domain.InquiryWithMessages, error) {
	if buyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if in.SellerID == uuid.Nil {
		return nil, apperrors.BadRequest("seller_id is required")
	}
	if in.SKUID == uuid.Nil {
		return nil, apperrors.BadRequest("sku_id is required")
	}
	in.Subject = strings.TrimSpace(in.Subject)
	in.InitialBody = strings.TrimSpace(in.InitialBody)
	if in.Subject == "" {
		return nil, apperrors.BadRequest("subject is required")
	}
	if in.InitialBody == "" {
		return nil, apperrors.BadRequest("initial_body is required")
	}
	if len(in.Subject) > 255 {
		return nil, apperrors.BadRequest("subject must be at most 255 characters")
	}
	if len(in.InitialBody) > 4000 {
		return nil, apperrors.BadRequest("initial_body must be at most 4000 characters")
	}

	// Purchase verification against the order service.
	check, err := s.orderClient.CheckPurchase(ctx, tenantID, buyerAuth0ID, in.SellerID, in.SKUID)
	if err != nil {
		return nil, err
	}
	if !check.Purchased {
		return nil, apperrors.Forbidden("you can only contact the seller of items you have purchased")
	}

	inq := &domain.Inquiry{
		BuyerAuth0ID: buyerAuth0ID,
		SellerID:     in.SellerID,
		SKUID:        in.SKUID,
		ProductName:  check.ProductName,
		SKUCode:      check.SKUCode,
		Subject:      in.Subject,
	}
	firstMsg := &domain.InquiryMessage{
		SenderType: domain.SenderTypeBuyer,
		SenderID:   buyerAuth0ID,
		Body:       in.InitialBody,
	}

	result, err := s.repo.Create(ctx, tenantID, inq, firstMsg)
	if err != nil {
		return nil, apperrors.Internal("failed to create inquiry", err)
	}

	s.publishMessageEvent(ctx, tenantID, &result.Inquiry, firstMsg)
	return result, nil
}

// PostMessage appends a reply to an existing thread after checking that
// the sender is a participant in it.
//
// Authorization is structural: buyers can post as `buyer` only when their
// Auth0 sub matches `inquiry.buyer_auth0_id`; sellers can post as `seller`
// only when their seller_id matches `inquiry.seller_id`.
func (s *InquiryService) PostMessage(
	ctx context.Context,
	tenantID uuid.UUID,
	actorAuth0ID string,
	actorSellerID *uuid.UUID,
	in domain.PostMessageInput,
) (*domain.InquiryMessage, error) {
	if in.InquiryID == uuid.Nil {
		return nil, apperrors.BadRequest("inquiry_id is required")
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		return nil, apperrors.BadRequest("body is required")
	}
	if len(in.Body) > 4000 {
		return nil, apperrors.BadRequest("body must be at most 4000 characters")
	}
	if in.SenderType != domain.SenderTypeBuyer && in.SenderType != domain.SenderTypeSeller {
		return nil, apperrors.BadRequest("invalid sender_type")
	}

	thread, err := s.repo.GetByID(ctx, tenantID, in.InquiryID)
	if err != nil {
		return nil, apperrors.Internal("failed to load inquiry", err)
	}
	if thread == nil {
		return nil, apperrors.NotFound("inquiry not found")
	}
	if thread.Status == domain.InquiryStatusClosed {
		return nil, apperrors.Forbidden("cannot post to a closed inquiry")
	}

	// Participant check.
	switch in.SenderType {
	case domain.SenderTypeBuyer:
		if actorAuth0ID == "" || actorAuth0ID != thread.BuyerAuth0ID {
			return nil, apperrors.Forbidden("not a participant of this inquiry")
		}
		in.SenderID = actorAuth0ID
	case domain.SenderTypeSeller:
		if actorSellerID == nil || *actorSellerID != thread.SellerID {
			return nil, apperrors.Forbidden("not a participant of this inquiry")
		}
		// We record the seller team member's auth0 sub as sender_id so we can
		// attribute replies when a seller team has multiple members.
		in.SenderID = actorAuth0ID
	}

	msg := &domain.InquiryMessage{
		InquiryID:  in.InquiryID,
		SenderType: in.SenderType,
		SenderID:   in.SenderID,
		Body:       in.Body,
	}
	if err := s.repo.AppendMessage(ctx, tenantID, msg); err != nil {
		return nil, apperrors.Internal("failed to append message", err)
	}

	s.publishMessageEvent(ctx, tenantID, &thread.Inquiry, msg)
	return msg, nil
}

// GetInquiry returns a thread plus its full message history after checking
// that the caller is either the buyer or a member of the seller team.
func (s *InquiryService) GetInquiry(
	ctx context.Context,
	tenantID, inquiryID uuid.UUID,
	actorAuth0ID string,
	actorSellerID *uuid.UUID,
) (*domain.InquiryWithMessages, error) {
	thread, err := s.repo.GetByID(ctx, tenantID, inquiryID)
	if err != nil {
		return nil, apperrors.Internal("failed to load inquiry", err)
	}
	if thread == nil {
		return nil, apperrors.NotFound("inquiry not found")
	}
	if !s.isParticipant(thread.Inquiry, actorAuth0ID, actorSellerID) {
		return nil, apperrors.NotFound("inquiry not found")
	}
	return thread, nil
}

// ListForBuyer returns the buyer's threads ordered by most recently active.
func (s *InquiryService) ListForBuyer(
	ctx context.Context,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	limit, offset int,
) ([]domain.Inquiry, int, error) {
	if buyerAuth0ID == "" {
		return nil, 0, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := s.repo.ListByBuyer(ctx, tenantID, buyerAuth0ID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list inquiries", err)
	}
	return items, total, nil
}

// ListForSeller returns a seller's received threads. status may be empty.
func (s *InquiryService) ListForSeller(
	ctx context.Context,
	tenantID, sellerID uuid.UUID,
	status string,
	limit, offset int,
) ([]domain.Inquiry, int, error) {
	if sellerID == uuid.Nil {
		return nil, 0, apperrors.BadRequest("seller_id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := s.repo.ListBySeller(ctx, tenantID, sellerID, status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list inquiries", err)
	}
	return items, total, nil
}

// MarkRead marks unread messages from the *other* party as read for the
// given reader role.
func (s *InquiryService) MarkRead(
	ctx context.Context,
	tenantID, inquiryID uuid.UUID,
	readerType string,
	actorAuth0ID string,
	actorSellerID *uuid.UUID,
) error {
	thread, err := s.repo.GetByID(ctx, tenantID, inquiryID)
	if err != nil {
		return apperrors.Internal("failed to load inquiry", err)
	}
	if thread == nil {
		return apperrors.NotFound("inquiry not found")
	}
	if !s.isParticipant(thread.Inquiry, actorAuth0ID, actorSellerID) {
		return apperrors.NotFound("inquiry not found")
	}
	if readerType != domain.SenderTypeBuyer && readerType != domain.SenderTypeSeller {
		return apperrors.BadRequest("invalid reader type")
	}
	if err := s.repo.MarkRead(ctx, tenantID, inquiryID, readerType); err != nil {
		return apperrors.Internal("failed to mark messages read", err)
	}
	return nil
}

// CloseInquiry transitions a thread to closed. Only the seller can close.
func (s *InquiryService) CloseInquiry(
	ctx context.Context,
	tenantID, inquiryID uuid.UUID,
	actorSellerID *uuid.UUID,
) error {
	thread, err := s.repo.GetByID(ctx, tenantID, inquiryID)
	if err != nil {
		return apperrors.Internal("failed to load inquiry", err)
	}
	if thread == nil {
		return apperrors.NotFound("inquiry not found")
	}
	if actorSellerID == nil || *actorSellerID != thread.SellerID {
		return apperrors.Forbidden("only the seller can close this inquiry")
	}
	if err := s.repo.Close(ctx, tenantID, inquiryID); err != nil {
		return apperrors.Internal("failed to close inquiry", err)
	}
	return nil
}

// isParticipant checks whether the actor is the buyer or a member of the
// seller team for the given thread.
func (s *InquiryService) isParticipant(
	inq domain.Inquiry,
	actorAuth0ID string,
	actorSellerID *uuid.UUID,
) bool {
	if actorAuth0ID != "" && actorAuth0ID == inq.BuyerAuth0ID {
		return true
	}
	if actorSellerID != nil && *actorSellerID == inq.SellerID {
		return true
	}
	return false
}

// publishMessageEvent is a best-effort publisher for inquiry.message_created.
// The topic is "inquiry-events"; the notification service subscribes to it
// and emails the opposite party. Failures are logged and swallowed — the
// user-facing request still succeeds if pubsub is unavailable.
func (s *InquiryService) publishMessageEvent(
	ctx context.Context,
	tenantID uuid.UUID,
	inq *domain.Inquiry,
	msg *domain.InquiryMessage,
) {
	preview := msg.Body
	if len(preview) > 200 {
		preview = preview[:200]
	}
	data := map[string]any{
		"inquiry_id":     inq.ID.String(),
		"seller_id":      inq.SellerID.String(),
		"buyer_auth0_id": inq.BuyerAuth0ID,
		"sender_type":    msg.SenderType,
		"subject":        inq.Subject,
		"product_name":   inq.ProductName,
		"body_preview":   preview,
	}
	pubsub.PublishEvent(ctx, s.publisher, tenantID, "inquiry.message_created", inquiryEventTopic, data)
}
