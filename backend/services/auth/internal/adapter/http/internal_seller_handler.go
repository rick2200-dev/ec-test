package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// SellerBatchGetUseCase is the minimal driving port InternalSellerHandler
// needs. Scoped narrowly so the handler does not pull in the full
// IdentityService surface.
type SellerBatchGetUseCase interface {
	BatchGetSellers(ctx context.Context, ids []uuid.UUID) ([]domain.Seller, error)
}

// InternalSellerHandler exposes batch seller lookups used by other services
// (order at checkout) to snapshot seller_name without reading auth_svc
// directly. Same shared-secret auth as other internal handlers.
type InternalSellerHandler struct {
	svc    SellerBatchGetUseCase
	secret string
}

// NewInternalSellerHandler constructs the handler. Empty secret rejects all
// requests (safe default).
func NewInternalSellerHandler(svc SellerBatchGetUseCase, secret string) *InternalSellerHandler {
	return &InternalSellerHandler{svc: svc, secret: secret}
}

// Routes returns the chi router for /internal/sellers endpoints.
func (h *InternalSellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.requireSharedSecret)
	r.Post("/batch-get", h.BatchGet)
	return r
}

func (h *InternalSellerHandler) requireSharedSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.secret == "" {
			httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "internal seller endpoint not configured"})
			return
		}
		if r.Header.Get("X-Internal-Token") != h.secret {
			httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type batchGetSellersRequest struct {
	IDs []string `json:"ids"`
}

type sellerSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type batchGetSellersResponse struct {
	Sellers []sellerSummary `json:"sellers"`
}

// BatchGet handles POST /internal/sellers/batch-get. Unknown ids are
// silently omitted — callers must tolerate length mismatch.
func (h *InternalSellerHandler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var req batchGetSellersRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id: " + raw})
			return
		}
		ids = append(ids, id)
	}
	sellers, err := h.svc.BatchGetSellers(r.Context(), ids)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	resp := batchGetSellersResponse{Sellers: make([]sellerSummary, 0, len(sellers))}
	for _, s := range sellers {
		resp.Sellers = append(resp.Sellers, sellerSummary{
			ID:   s.ID.String(),
			Name: s.Name,
			Slug: s.Slug,
		})
	}
	httputil.JSON(w, http.StatusOK, resp)
}
