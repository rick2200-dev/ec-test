package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	app "github.com/Riku-KANO/ec-test/services/inquiry/internal/app"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/port"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type mockStore struct {
	createFn        func(ctx context.Context, tenantID uuid.UUID, inq *domain.Inquiry, msg *domain.InquiryMessage) (*domain.InquiryWithMessages, error)
	getByIDFn       func(ctx context.Context, tenantID, inquiryID uuid.UUID) (*domain.InquiryWithMessages, error)
	listByBuyerFn   func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error)
	listBySellerFn  func(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error)
	appendMessageFn func(ctx context.Context, tenantID uuid.UUID, msg *domain.InquiryMessage) error
	markReadFn      func(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string) error
	closeFn         func(ctx context.Context, tenantID, inquiryID uuid.UUID) error
}

func (m *mockStore) Create(ctx context.Context, tenantID uuid.UUID, inq *domain.Inquiry, msg *domain.InquiryMessage) (*domain.InquiryWithMessages, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, inq, msg)
	}
	return &domain.InquiryWithMessages{Inquiry: *inq, Messages: []domain.InquiryMessage{*msg}}, nil
}

func (m *mockStore) GetByID(ctx context.Context, tenantID, inquiryID uuid.UUID) (*domain.InquiryWithMessages, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, inquiryID)
	}
	return nil, nil
}

func (m *mockStore) ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Inquiry, int, error) {
	if m.listByBuyerFn != nil {
		return m.listByBuyerFn(ctx, tenantID, buyerAuth0ID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockStore) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Inquiry, int, error) {
	if m.listBySellerFn != nil {
		return m.listBySellerFn(ctx, tenantID, sellerID, status, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockStore) AppendMessage(ctx context.Context, tenantID uuid.UUID, msg *domain.InquiryMessage) error {
	if m.appendMessageFn != nil {
		return m.appendMessageFn(ctx, tenantID, msg)
	}
	return nil
}

func (m *mockStore) MarkRead(ctx context.Context, tenantID, inquiryID uuid.UUID, readerType string) error {
	if m.markReadFn != nil {
		return m.markReadFn(ctx, tenantID, inquiryID, readerType)
	}
	return nil
}

func (m *mockStore) Close(ctx context.Context, tenantID, inquiryID uuid.UUID) error {
	if m.closeFn != nil {
		return m.closeFn(ctx, tenantID, inquiryID)
	}
	return nil
}

type mockPurchaseChecker struct {
	checkFn func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error)
}

func (m *mockPurchaseChecker) CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error) {
	if m.checkFn != nil {
		return m.checkFn(ctx, tenantID, buyerAuth0ID, sellerID, skuID)
	}
	return &port.PurchaseCheckResult{
		Purchased:   true,
		ProductName: "Test Product",
		SKUCode:     "SKU-001",
	}, nil
}

// Compile-time interface checks.
var _ port.InquiryStore = (*mockStore)(nil)
var _ port.PurchaseChecker = (*mockPurchaseChecker)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newService(store *mockStore, checker *mockPurchaseChecker) *app.InquiryService {
	return app.NewInquiryService(store, checker, nil)
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }

func validCreateInput() domain.CreateInquiryInput {
	return domain.CreateInquiryInput{
		SellerID:    uuid.New(),
		SKUID:       uuid.New(),
		Subject:     "Shipping question",
		InitialBody: "When will my order arrive?",
	}
}

// ---------------------------------------------------------------------------
// CreateInquiry
// ---------------------------------------------------------------------------

func TestCreateInquiry_Success(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	checker := &mockPurchaseChecker{}
	svc := newService(store, checker)

	tenantID := uuid.New()
	buyerAuth0ID := "auth0|buyer1"
	in := validCreateInput()

	result, err := svc.CreateInquiry(context.Background(), tenantID, buyerAuth0ID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BuyerAuth0ID != buyerAuth0ID {
		t.Errorf("BuyerAuth0ID = %q, want %q", result.BuyerAuth0ID, buyerAuth0ID)
	}
	if result.SellerID != in.SellerID {
		t.Errorf("SellerID = %v, want %v", result.SellerID, in.SellerID)
	}
	if result.Subject != in.Subject {
		t.Errorf("Subject = %q, want %q", result.Subject, in.Subject)
	}
	if result.ProductName != "Test Product" {
		t.Errorf("ProductName = %q, want %q", result.ProductName, "Test Product")
	}
	if result.SKUCode != "SKU-001" {
		t.Errorf("SKUCode = %q, want %q", result.SKUCode, "SKU-001")
	}
	if len(result.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].Body != in.InitialBody {
		t.Errorf("Messages[0].Body = %q, want %q", result.Messages[0].Body, in.InitialBody)
	}
}

func TestCreateInquiry_EmptyBuyer(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "", validCreateInput())
	if err == nil {
		t.Fatal("expected error for empty buyer_auth0_id")
	}
	if !containsMessage(err, "buyer_auth0_id is required") {
		t.Errorf("error = %v, want message containing 'buyer_auth0_id is required'", err)
	}
}

func TestCreateInquiry_EmptySellerID(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})
	in := validCreateInput()
	in.SellerID = uuid.Nil

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "auth0|buyer1", in)
	if err == nil {
		t.Fatal("expected error for nil seller_id")
	}
	if !containsMessage(err, "seller_id is required") {
		t.Errorf("error = %v, want message containing 'seller_id is required'", err)
	}
}

func TestCreateInquiry_EmptySKUID(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})
	in := validCreateInput()
	in.SKUID = uuid.Nil

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "auth0|buyer1", in)
	if err == nil {
		t.Fatal("expected error for nil sku_id")
	}
	if !containsMessage(err, "sku_id is required") {
		t.Errorf("error = %v, want message containing 'sku_id is required'", err)
	}
}

func TestCreateInquiry_EmptySubject(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})
	in := validCreateInput()
	in.Subject = ""

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "auth0|buyer1", in)
	if err == nil {
		t.Fatal("expected error for empty subject")
	}
	if !containsMessage(err, "subject is required") {
		t.Errorf("error = %v, want message containing 'subject is required'", err)
	}
}

func TestCreateInquiry_EmptyBody(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})
	in := validCreateInput()
	in.InitialBody = ""

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "auth0|buyer1", in)
	if err == nil {
		t.Fatal("expected error for empty initial_body")
	}
	if !containsMessage(err, "initial_body is required") {
		t.Errorf("error = %v, want message containing 'initial_body is required'", err)
	}
}

func TestCreateInquiry_PurchaseNotFound(t *testing.T) {
	t.Parallel()

	checker := &mockPurchaseChecker{
		checkFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			return &port.PurchaseCheckResult{Purchased: false}, nil
		},
	}
	svc := newService(&mockStore{}, checker)

	_, err := svc.CreateInquiry(context.Background(), uuid.New(), "auth0|buyer1", validCreateInput())
	if !errors.Is(err, domain.ErrPurchaseRequired) {
		t.Errorf("error = %v, want ErrPurchaseRequired", err)
	}
}

// ---------------------------------------------------------------------------
// PostMessage
// ---------------------------------------------------------------------------

func TestPostMessage_SuccessAsBuyer(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	sellerID := uuid.New()
	inquiryID := uuid.New()
	tenantID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					TenantID:     tenantID,
					BuyerAuth0ID: buyerAuth0ID,
					SellerID:     sellerID,
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  inquiryID,
		SenderType: domain.SenderTypeBuyer,
		Body:       "follow-up question",
	}

	msg, err := svc.PostMessage(context.Background(), tenantID, buyerAuth0ID, nil, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.SenderType != domain.SenderTypeBuyer {
		t.Errorf("SenderType = %q, want %q", msg.SenderType, domain.SenderTypeBuyer)
	}
	if msg.SenderID != buyerAuth0ID {
		t.Errorf("SenderID = %q, want %q", msg.SenderID, buyerAuth0ID)
	}
	if msg.Body != "follow-up question" {
		t.Errorf("Body = %q, want %q", msg.Body, "follow-up question")
	}
}

func TestPostMessage_SuccessAsSeller(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	sellerAuth0ID := "auth0|seller1"
	sellerID := uuid.New()
	inquiryID := uuid.New()
	tenantID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					TenantID:     tenantID,
					BuyerAuth0ID: buyerAuth0ID,
					SellerID:     sellerID,
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  inquiryID,
		SenderType: domain.SenderTypeSeller,
		Body:       "seller reply",
	}

	msg, err := svc.PostMessage(context.Background(), tenantID, sellerAuth0ID, ptrUUID(sellerID), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.SenderType != domain.SenderTypeSeller {
		t.Errorf("SenderType = %q, want %q", msg.SenderType, domain.SenderTypeSeller)
	}
	if msg.SenderID != sellerAuth0ID {
		t.Errorf("SenderID = %q, want %q (seller team member auth0 sub)", msg.SenderID, sellerAuth0ID)
	}
}

func TestPostMessage_InvalidSenderType(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  uuid.New(),
		SenderType: "admin",
		Body:       "hello",
	}

	_, err := svc.PostMessage(context.Background(), uuid.New(), "auth0|x", nil, in)
	if !errors.Is(err, domain.ErrInvalidSenderType) {
		t.Errorf("error = %v, want ErrInvalidSenderType", err)
	}
}

func TestPostMessage_InquiryNotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return nil, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  uuid.New(),
		SenderType: domain.SenderTypeBuyer,
		Body:       "hello",
	}

	_, err := svc.PostMessage(context.Background(), uuid.New(), "auth0|buyer1", nil, in)
	if !errors.Is(err, domain.ErrInquiryNotFound) {
		t.Errorf("error = %v, want ErrInquiryNotFound", err)
	}
}

func TestPostMessage_InquiryClosed(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	inquiryID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					BuyerAuth0ID: buyerAuth0ID,
					SellerID:     uuid.New(),
					Status:       domain.InquiryStatusClosed,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  inquiryID,
		SenderType: domain.SenderTypeBuyer,
		Body:       "hello",
	}

	_, err := svc.PostMessage(context.Background(), uuid.New(), buyerAuth0ID, nil, in)
	if !errors.Is(err, domain.ErrInquiryClosed) {
		t.Errorf("error = %v, want ErrInquiryClosed", err)
	}
}

func TestPostMessage_NotParticipant(t *testing.T) {
	t.Parallel()

	inquiryID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					BuyerAuth0ID: "auth0|the-real-buyer",
					SellerID:     uuid.New(),
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	in := domain.PostMessageInput{
		InquiryID:  inquiryID,
		SenderType: domain.SenderTypeBuyer,
		Body:       "hello",
	}

	// An actor whose auth0 id does not match the buyer on the inquiry.
	_, err := svc.PostMessage(context.Background(), uuid.New(), "auth0|imposter", nil, in)
	if !errors.Is(err, domain.ErrNotParticipant) {
		t.Errorf("error = %v, want ErrNotParticipant", err)
	}
}

// ---------------------------------------------------------------------------
// GetInquiry
// ---------------------------------------------------------------------------

func TestGetInquiry_Success(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	sellerID := uuid.New()
	inquiryID := uuid.New()
	tenantID := uuid.New()

	thread := &domain.InquiryWithMessages{
		Inquiry: domain.Inquiry{
			ID:           inquiryID,
			TenantID:     tenantID,
			BuyerAuth0ID: buyerAuth0ID,
			SellerID:     sellerID,
			Status:       domain.InquiryStatusOpen,
			Subject:      "question",
		},
		Messages: []domain.InquiryMessage{
			{ID: uuid.New(), Body: "first message"},
		},
	}

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return thread, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	// Buyer access.
	result, err := svc.GetInquiry(context.Background(), tenantID, inquiryID, buyerAuth0ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != inquiryID {
		t.Errorf("ID = %v, want %v", result.ID, inquiryID)
	}
	if len(result.Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1", len(result.Messages))
	}

	// Seller access.
	result2, err2 := svc.GetInquiry(context.Background(), tenantID, inquiryID, "", ptrUUID(sellerID))
	if err2 != nil {
		t.Fatalf("unexpected error for seller access: %v", err2)
	}
	if result2.ID != inquiryID {
		t.Errorf("ID = %v, want %v", result2.ID, inquiryID)
	}
}

func TestGetInquiry_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return nil, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	_, err := svc.GetInquiry(context.Background(), uuid.New(), uuid.New(), "auth0|buyer1", nil)
	if !errors.Is(err, domain.ErrInquiryNotFound) {
		t.Errorf("error = %v, want ErrInquiryNotFound", err)
	}
}

func TestGetInquiry_NotParticipant(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           uuid.New(),
					BuyerAuth0ID: "auth0|the-real-buyer",
					SellerID:     uuid.New(),
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	// Neither buyer nor seller matches.
	_, err := svc.GetInquiry(context.Background(), uuid.New(), uuid.New(), "auth0|stranger", ptrUUID(uuid.New()))
	// The service returns ErrInquiryNotFound (rather than ErrNotParticipant) to
	// avoid leaking existence of the thread to non-participants.
	if !errors.Is(err, domain.ErrInquiryNotFound) {
		t.Errorf("error = %v, want ErrInquiryNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// ListForBuyer
// ---------------------------------------------------------------------------

func TestListForBuyer_Success(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	tenantID := uuid.New()
	expected := []domain.Inquiry{
		{ID: uuid.New(), BuyerAuth0ID: buyerAuth0ID, Subject: "q1"},
		{ID: uuid.New(), BuyerAuth0ID: buyerAuth0ID, Subject: "q2"},
	}

	store := &mockStore{
		listByBuyerFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]domain.Inquiry, int, error) {
			return expected, 2, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	items, total, err := svc.ListForBuyer(context.Background(), tenantID, buyerAuth0ID, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestListForBuyer_EmptyBuyerAuth0ID(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForBuyer(context.Background(), uuid.New(), "", 20, 0)
	if err == nil {
		t.Fatal("expected error for empty buyer_auth0_id")
	}
	if !containsMessage(err, "buyer_auth0_id is required") {
		t.Errorf("error = %v, want message containing 'buyer_auth0_id is required'", err)
	}
}

// ---------------------------------------------------------------------------
// ListForSeller
// ---------------------------------------------------------------------------

func TestListForSeller_Success(t *testing.T) {
	t.Parallel()

	sellerID := uuid.New()
	tenantID := uuid.New()
	expected := []domain.Inquiry{
		{ID: uuid.New(), SellerID: sellerID, Subject: "q1"},
	}

	store := &mockStore{
		listBySellerFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _, _ int) ([]domain.Inquiry, int, error) {
			return expected, 1, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	items, total, err := svc.ListForSeller(context.Background(), tenantID, sellerID, "", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestListForSeller_NilSellerID(t *testing.T) {
	t.Parallel()

	svc := newService(&mockStore{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForSeller(context.Background(), uuid.New(), uuid.Nil, "", 20, 0)
	if err == nil {
		t.Fatal("expected error for nil seller_id")
	}
	if !containsMessage(err, "seller_id is required") {
		t.Errorf("error = %v, want message containing 'seller_id is required'", err)
	}
}

// ---------------------------------------------------------------------------
// MarkRead
// ---------------------------------------------------------------------------

func TestMarkRead_Success(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	sellerID := uuid.New()
	inquiryID := uuid.New()
	tenantID := uuid.New()
	markReadCalled := false

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					TenantID:     tenantID,
					BuyerAuth0ID: buyerAuth0ID,
					SellerID:     sellerID,
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
		markReadFn: func(_ context.Context, _, _ uuid.UUID, _ string) error {
			markReadCalled = true
			return nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.MarkRead(context.Background(), tenantID, inquiryID, domain.SenderTypeBuyer, buyerAuth0ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !markReadCalled {
		t.Error("expected MarkRead to be called on the store")
	}
}

func TestMarkRead_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return nil, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.MarkRead(context.Background(), uuid.New(), uuid.New(), domain.SenderTypeBuyer, "auth0|buyer1", nil)
	if !errors.Is(err, domain.ErrInquiryNotFound) {
		t.Errorf("error = %v, want ErrInquiryNotFound", err)
	}
}

func TestMarkRead_InvalidReaderType(t *testing.T) {
	t.Parallel()

	buyerAuth0ID := "auth0|buyer1"
	inquiryID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					BuyerAuth0ID: buyerAuth0ID,
					SellerID:     uuid.New(),
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.MarkRead(context.Background(), uuid.New(), inquiryID, "admin", buyerAuth0ID, nil)
	if !errors.Is(err, domain.ErrInvalidReaderType) {
		t.Errorf("error = %v, want ErrInvalidReaderType", err)
	}
}

// ---------------------------------------------------------------------------
// CloseInquiry
// ---------------------------------------------------------------------------

func TestCloseInquiry_Success(t *testing.T) {
	t.Parallel()

	sellerID := uuid.New()
	inquiryID := uuid.New()
	tenantID := uuid.New()
	closeCalled := false

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					TenantID:     tenantID,
					BuyerAuth0ID: "auth0|buyer1",
					SellerID:     sellerID,
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
		closeFn: func(_ context.Context, _, _ uuid.UUID) error {
			closeCalled = true
			return nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.CloseInquiry(context.Background(), tenantID, inquiryID, ptrUUID(sellerID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closeCalled {
		t.Error("expected Close to be called on the store")
	}
}

func TestCloseInquiry_NotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return nil, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.CloseInquiry(context.Background(), uuid.New(), uuid.New(), ptrUUID(uuid.New()))
	if !errors.Is(err, domain.ErrInquiryNotFound) {
		t.Errorf("error = %v, want ErrInquiryNotFound", err)
	}
}

func TestCloseInquiry_NilSellerID(t *testing.T) {
	t.Parallel()

	inquiryID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					BuyerAuth0ID: "auth0|buyer1",
					SellerID:     uuid.New(),
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	err := svc.CloseInquiry(context.Background(), uuid.New(), inquiryID, nil)
	if !errors.Is(err, domain.ErrNotParticipant) {
		t.Errorf("error = %v, want ErrNotParticipant", err)
	}
}

func TestCloseInquiry_WrongSellerID(t *testing.T) {
	t.Parallel()

	inquiryID := uuid.New()

	store := &mockStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.InquiryWithMessages, error) {
			return &domain.InquiryWithMessages{
				Inquiry: domain.Inquiry{
					ID:           inquiryID,
					BuyerAuth0ID: "auth0|buyer1",
					SellerID:     uuid.New(), // the real seller
					Status:       domain.InquiryStatusOpen,
				},
			}, nil
		},
	}
	svc := newService(store, &mockPurchaseChecker{})

	wrongSeller := uuid.New()
	err := svc.CloseInquiry(context.Background(), uuid.New(), inquiryID, ptrUUID(wrongSeller))
	if !errors.Is(err, domain.ErrNotParticipant) {
		t.Errorf("error = %v, want ErrNotParticipant", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers (test utilities)
// ---------------------------------------------------------------------------

// containsMessage checks whether err's message contains the given substring.
// Works with *apperrors.AppError and plain errors.
func containsMessage(err error, substr string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) > 0 && contains(msg, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
