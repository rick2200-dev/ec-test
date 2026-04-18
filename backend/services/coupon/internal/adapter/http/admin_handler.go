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

// AdminHandler exposes the platform-admin CRUD endpoints. Gateway
// mounts these under /api/v1/admin/coupons/* with platform-admin
// authorization applied at the gateway layer — the coupon service only
// verifies the X-Internal-Token.
type AdminHandler struct {
	svc port.CouponUseCase
}

func NewAdminHandler(svc port.CouponUseCase) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/revoke", h.Revoke)
	r.Get("/{id}/stats", h.Stats)
	return r
}

type createCouponRequest struct {
	Code               string  `json:"code"`
	IssuerType         string  `json:"issuer_type"`
	IssuerID           *string `json:"issuer_id,omitempty"`
	DiscountType       string  `json:"discount_type"`
	DiscountPercentBps *int    `json:"discount_percent_bps,omitempty"`
	DiscountAmount     *int64  `json:"discount_amount,omitempty"`
	Currency           string  `json:"currency"`
	MinOrderAmount     int64   `json:"min_order_amount"`
	MaxDiscountAmount  *int64  `json:"max_discount_amount,omitempty"`
	ValidFromUnix      *int64  `json:"valid_from_unix,omitempty"`
	ExpiresAtUnix      *int64  `json:"expires_at_unix,omitempty"`
	UsageLimitTotal    *int    `json:"usage_limit_total,omitempty"`
	UsageLimitPerUser  *int    `json:"usage_limit_per_user,omitempty"`
	Title              string  `json:"title"`
	Description        string  `json:"description"`
}

func (h *AdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCouponRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	input := port.CreateCouponInput{
		Code:               req.Code,
		IssuerType:         domain.IssuerType(defaultString(req.IssuerType, string(domain.IssuerTypePlatform))),
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
	if req.IssuerID != nil && *req.IssuerID != "" {
		id, err := uuid.Parse(*req.IssuerID)
		if err != nil {
			httputil.Error(w, apperrors.BadRequest("issuer_id is not a valid UUID"))
			return
		}
		input.IssuerID = &id
	}
	coupon, err := h.svc.CreateCoupon(r.Context(), input)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, coupon)
}

type listCouponsResponse struct {
	Coupons []domain.Coupon `json:"coupons"`
	Total   int             `json:"total"`
}

func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	coupons, total, err := h.svc.ListCoupons(r.Context(), port.ListCouponsFilter{
		Status: r.URL.Query().Get("status"),
		Limit:  limit,
		Offset: offset,
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

func (h *AdminHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	coupon, err := h.svc.GetCoupon(r.Context(), id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, coupon)
}

func (h *AdminHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	coupon, err := h.svc.RevokeCoupon(r.Context(), id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, coupon)
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("id is not a valid UUID"))
		return
	}
	stats, err := h.svc.GetCouponStats(r.Context(), id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}

func defaultString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
