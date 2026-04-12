package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
	"github.com/Riku-KANO/ec-test/services/review/internal/port"
)

// BuyerHandler exposes the buyer-facing review REST API.
type BuyerHandler struct {
	svc port.ReviewUseCase
}

// NewBuyerHandler constructs a BuyerHandler.
func NewBuyerHandler(svc port.ReviewUseCase) *BuyerHandler {
	return &BuyerHandler{svc: svc}
}

// Routes returns the chi router for /reviews (buyer scope).
func (h *BuyerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/product/{productId}", h.ListByProduct)
	r.Get("/product/{productId}/rating", h.GetProductRating)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// createReviewRequest is the POST /reviews payload.
type createReviewRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Rating    int       `json:"rating"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
}

// Create handles POST /reviews.
func (h *BuyerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	var req createReviewRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	review, err := h.svc.CreateReview(r.Context(), tc.TenantID, tc.UserID, domain.CreateReviewInput{
		ProductID: req.ProductID,
		Rating:    req.Rating,
		Title:     req.Title,
		Body:      req.Body,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, review)
}

// updateReviewRequest is the PUT /reviews/{id} payload.
type updateReviewRequest struct {
	Rating *int    `json:"rating,omitempty"`
	Title  *string `json:"title,omitempty"`
	Body   *string `json:"body,omitempty"`
}

// Update handles PUT /reviews/{id}.
func (h *BuyerHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}
	var req updateReviewRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	review, err := h.svc.UpdateReview(r.Context(), tc.TenantID, id, tc.UserID, domain.UpdateReviewInput{
		Rating: req.Rating,
		Title:  req.Title,
		Body:   req.Body,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, review)
}

// Delete handles DELETE /reviews/{id}.
func (h *BuyerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}

	if err := h.svc.DeleteReview(r.Context(), tc.TenantID, id, tc.UserID); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Get handles GET /reviews/{id}.
func (h *BuyerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}

	review, err := h.svc.GetReview(r.Context(), tc.TenantID, id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, review)
}

// ListByProduct handles GET /reviews/product/{productId}.
func (h *BuyerHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid product id"))
		return
	}
	limit, offset := parsePagination(r)

	items, total, err := h.svc.ListByProduct(r.Context(), tc.TenantID, productID, limit, offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, reviewListResponse{Items: items, Total: total, Limit: limit, Offset: offset})
}

// GetProductRating handles GET /reviews/product/{productId}/rating.
func (h *BuyerHandler) GetProductRating(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid product id"))
		return
	}

	rating, err := h.svc.GetProductRating(r.Context(), tc.TenantID, productID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, rating)
}

// reviewListResponse is the standard paginated list envelope for reviews.
type reviewListResponse struct {
	Items  []domain.Review `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// parsePagination reads limit/offset from query params with sensible defaults.
func parsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
