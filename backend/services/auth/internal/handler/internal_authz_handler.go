package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
)

// InternalAuthzHandler exposes role lookup endpoints used by the API gateway
// to enforce fine-grained authorization on requests that have already passed
// JWT validation. These endpoints are protected by a shared secret and MUST
// NOT be exposed to end users.
type InternalAuthzHandler struct {
	svc    *service.AuthService
	secret string
}

// NewInternalAuthzHandler creates a new InternalAuthzHandler. If secret is
// empty the handler rejects all requests (safe default).
func NewInternalAuthzHandler(svc *service.AuthService, secret string) *InternalAuthzHandler {
	return &InternalAuthzHandler{svc: svc, secret: secret}
}

// Routes returns the chi router for internal authz endpoints, protected by
// the shared-secret middleware.
func (h *InternalAuthzHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.requireSharedSecret)
	r.Get("/seller-role", h.GetSellerRole)
	r.Get("/platform-admin-role", h.GetPlatformAdminRole)
	// API token lookup is POST so the secret portion is in the body (never
	// in an access log or URL). The gateway calls this on every uncached
	// API-token request.
	r.Post("/api-token", h.LookupAPIToken)
	return r
}

func (h *InternalAuthzHandler) requireSharedSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.secret == "" {
			httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "internal authz not configured"})
			return
		}
		if r.Header.Get("X-Internal-Token") != h.secret {
			httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid internal token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type sellerRoleResponse struct {
	Role string `json:"role"`
}

// GetSellerRole handles GET /internal/authz/seller-role?seller_id=&sub=
// Tenant ID is taken from the X-Tenant-ID header set via InternalContext
// middleware. Returns the role as a JSON string; empty if not a member.
func (h *InternalAuthzHandler) GetSellerRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "missing tenant id"})
		return
	}
	q := r.URL.Query()
	sellerIDStr := q.Get("seller_id")
	sub := q.Get("sub")
	if sellerIDStr == "" || sub == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "seller_id and sub are required"})
		return
	}
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller_id"})
		return
	}
	role, err := h.svc.LookupSellerRole(r.Context(), tenantID, sellerID, sub)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, sellerRoleResponse{Role: string(role)})
}

// GetPlatformAdminRole handles GET /internal/authz/platform-admin-role?sub=
func (h *InternalAuthzHandler) GetPlatformAdminRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "missing tenant id"})
		return
	}
	sub := r.URL.Query().Get("sub")
	if sub == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "sub is required"})
		return
	}
	role, err := h.svc.LookupPlatformAdminRole(r.Context(), tenantID, sub)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, sellerRoleResponse{Role: string(role)})
}

// apiTokenLookupRequest is the body for POST /internal/authz/api-token.
// The three pieces are handed over separately so the gateway can fail fast
// on malformed payloads without this handler needing to parse the wire
// format itself.
type apiTokenLookupRequest struct {
	Prefix string `json:"prefix"`
	Lookup string `json:"lookup"`
	Secret string `json:"secret"`
}

// apiTokenLookupResponse is the JSON payload returned on a successful
// lookup. Status is one of "active", "revoked", "expired", "not_found",
// "invalid". Only "active" responses include a Token field.
type apiTokenLookupResponse struct {
	Status string           `json:"status"`
	Token  *apiTokenSummary `json:"token,omitempty"`
}

// apiTokenSummary mirrors the fields the gateway needs to build its
// tenant.Context and apply rate limiting. Sensitive columns (token_hash
// in particular) are never included.
type apiTokenSummary struct {
	ID                  uuid.UUID `json:"id"`
	TenantID            uuid.UUID `json:"tenant_id"`
	SellerID            uuid.UUID `json:"seller_id"`
	Scopes              []string  `json:"scopes"`
	RateLimitRPS        *int      `json:"rate_limit_rps,omitempty"`
	RateLimitBurst      *int      `json:"rate_limit_burst,omitempty"`
	IssuedByAuth0UserID string    `json:"issued_by_auth0_user_id"`
}

// LookupAPIToken handles POST /internal/authz/api-token. It verifies the
// supplied token pieces against the database (constant-time hash compare
// in the service layer) and returns a compact summary used by the gateway
// to populate its request context. A 200 response with status != "active"
// means the token exists but cannot be used; the gateway MUST translate
// those to 401 for the caller.
func (h *InternalAuthzHandler) LookupAPIToken(w http.ResponseWriter, r *http.Request) {
	var req apiTokenLookupRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if req.Prefix == "" || req.Lookup == "" || req.Secret == "" {
		httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{Status: "invalid"})
		return
	}

	tok, err := h.svc.LookupAPIToken(r.Context(), req.Prefix, req.Lookup, req.Secret)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAPITokenNotFound):
			httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{Status: "not_found"})
		case errors.Is(err, domain.ErrAPITokenRevoked):
			httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{Status: "revoked"})
		case errors.Is(err, domain.ErrAPITokenExpired):
			httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{Status: "expired"})
		case errors.Is(err, domain.ErrAPITokenInvalidFormat):
			httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{Status: "invalid"})
		default:
			httputil.Error(w, err)
		}
		return
	}

	scopeStrings := make([]string, len(tok.Scopes))
	for i, sc := range tok.Scopes {
		scopeStrings[i] = string(sc)
	}
	httputil.JSON(w, http.StatusOK, apiTokenLookupResponse{
		Status: "active",
		Token: &apiTokenSummary{
			ID:                  tok.ID,
			TenantID:            tok.TenantID,
			SellerID:            tok.SellerID,
			Scopes:              scopeStrings,
			RateLimitRPS:        tok.RateLimitRPS,
			RateLimitBurst:      tok.RateLimitBurst,
			IssuedByAuth0UserID: tok.IssuedByAuth0UserID,
		},
	})
}
