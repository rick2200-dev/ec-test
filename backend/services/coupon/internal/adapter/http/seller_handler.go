package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// SellerHandler exposes the seller-facing coupon management endpoints.
// The gateway proxies /api/v1/seller/coupons/* here; the authenticated
// seller UUID arrives as the X-Seller-ID header (set by the gateway
// from the caller's tenant context). All writes here force IssuerType
// = seller and IssuerID = X-Seller-ID so a compromised request body
// can't mint a coupon owned by a different seller.
type SellerHandler struct {
	svc port.CouponUseCase
}

func NewSellerHandler(svc port.CouponUseCase) *SellerHandler {
	return &SellerHandler{svc: svc}
}

func (h *SellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Post("/{id}/revoke", h.Revoke)
	r.Get("/{id}/stats", h.Stats)
	return r
}

func (h *SellerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	// Same ownership guard as Get / Revoke — prevents seller A from
	// editing seller B's coupons even if they guess the UUID.
	if _, err := h.loadOwned(r, id); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	var req updateCouponRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	updated, err := h.svc.UpdateCoupon(r.Context(), id, req.toInput())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, updated)
}

func (h *SellerHandler) sellerID(r *http.Request) (uuid.UUID, error) {
	raw := r.Header.Get("X-Seller-ID")
	if raw == "" {
		return uuid.Nil, apperrors.Unauthorized("missing X-Seller-ID")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.BadRequest("X-Seller-ID is not a valid UUID")
	}
	return id, nil
}

func (h *SellerHandler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	var req createCouponRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	// Seller requests cannot mint platform coupons or coupons owned by
	// another seller — force the issuer fields regardless of what the
	// client sent.
	input := port.CreateCouponInput{
		Code:               req.Code,
		IssuerType:         domain.IssuerTypeSeller,
		IssuerID:           &sellerID,
		DiscountType:       domain.DiscountType(req.DiscountType),
		DiscountPercentBps: req.DiscountPercentBps,
		DiscountAmount:     req.DiscountAmount,
		Currency:           req.Currency,
		MinOrderAmount:     req.MinOrderAmount,
		MaxDiscountAmount:  req.MaxDiscountAmount,
		ExpiresAtUnix:      req.ExpiresAtUnix,
		UsageLimitTotal:    req.UsageLimitTotal,
		UsageLimitPerUser:  req.UsageLimitPerUser,
		Title:              req.Title,
		Description:        req.Description,
	}
	if req.ValidFromUnix != nil {
		input.ValidFromSet = true
		input.ValidFromUnix = *req.ValidFromUnix
	}
	coupon, err := h.svc.CreateCoupon(r.Context(), input)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, coupon)
}

func (h *SellerHandler) List(w http.ResponseWriter, r *http.Request) {
	sellerID, err := h.sellerID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	coupons, total, err := h.svc.ListCoupons(r.Context(), port.ListCouponsFilter{
		Status:     r.URL.Query().Get("status"),
		IssuerType: domain.IssuerTypeSeller,
		IssuerID:   &sellerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if coupons == nil {
		coupons = []domain.Coupon{}
	}
	httputil.JSON(w, http.StatusOK, listCouponsResponse{Coupons: coupons, Total: total})
}

// loadOwned fetches a coupon and returns 404 unless it belongs to the
// calling seller. Keeps seller A from probing for seller B's coupon IDs
// by brute force.
func (h *SellerHandler) loadOwned(r *http.Request, id uuid.UUID) (*domain.Coupon, error) {
	sellerID, err := h.sellerID(r)
	if err != nil {
		return nil, err
	}
	c, err := h.svc.GetCoupon(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if c.IssuerType != domain.IssuerTypeSeller || c.IssuerID == nil || *c.IssuerID != sellerID {
		return nil, apperrors.NotFound("coupon not found").WithCode("COUPON_NOT_FOUND")
	}
	return c, nil
}

func (h *SellerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	c, err := h.loadOwned(r, id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, c)
}

func (h *SellerHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	if _, err := h.loadOwned(r, id); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	coupon, err := h.svc.RevokeCoupon(r.Context(), id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, coupon)
}

func (h *SellerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	if _, err := h.loadOwned(r, id); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	stats, err := h.svc.GetCouponStats(r.Context(), id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}
