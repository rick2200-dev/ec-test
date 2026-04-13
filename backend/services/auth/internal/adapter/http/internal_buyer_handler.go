package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// BuyerUpsertUseCase is the driving port required by InternalBuyerHandler.
// Kept local so the handler is not coupled to the full BuyerService struct.
type BuyerUpsertUseCase interface {
	UpsertBuyer(ctx context.Context, tenantID uuid.UUID, auth0Sub, email, displayName string) (*domain.Buyer, error)
}

// InternalBuyerHandler exposes the buyer profile upsert endpoint used by
// the Next.js BFF during Auth0 `onCallback`. Same shared-secret pattern as
// InternalAuthzHandler — must NOT be exposed to end users.
type InternalBuyerHandler struct {
	svc    BuyerUpsertUseCase
	secret string
}

// NewInternalBuyerHandler constructs the handler. An empty secret rejects
// all requests (safe default).
func NewInternalBuyerHandler(svc BuyerUpsertUseCase, secret string) *InternalBuyerHandler {
	return &InternalBuyerHandler{svc: svc, secret: secret}
}

// Routes returns the chi router for /internal/buyers endpoints.
func (h *InternalBuyerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.requireSharedSecret)
	r.Post("/upsert", h.Upsert)
	return r
}

func (h *InternalBuyerHandler) requireSharedSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.secret == "" {
			httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "internal buyer endpoint not configured"})
			return
		}
		if r.Header.Get("X-Internal-Token") != h.secret {
			httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// buyerUpsertRequest is the JSON body for POST /internal/buyers/upsert.
// tenant_id is included explicitly because the BFF knows it from env
// (single-tenant MVP) — this keeps the handler stateless.
type buyerUpsertRequest struct {
	TenantID    string `json:"tenant_id"`
	Auth0Sub    string `json:"auth0_sub"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
}

// Upsert handles POST /internal/buyers/upsert.
func (h *InternalBuyerHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var req buyerUpsertRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
		return
	}
	buyer, err := h.svc.UpsertBuyer(r.Context(), tenantID, req.Auth0Sub, req.Email, req.DisplayName)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, buyer)
}
