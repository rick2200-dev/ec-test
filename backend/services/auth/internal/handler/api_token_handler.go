package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
)

// APITokenHandler handles HTTP requests for seller API access token
// management. Mounted at /sellers/{sellerID}/api-tokens from inside the
// seller router, parallel to the existing team subtree.
//
// NOTE: this surface is UI-only — it MUST NOT be callable with an API
// token itself (privilege escalation: a token rewriting its own scopes
// or issuing a stronger sibling token). The gateway enforces this via
// BlockAPIToken middleware on the /api-tokens subtree.
type APITokenHandler struct {
	svc    *service.AuthService
	prefix string
}

// NewAPITokenHandler creates a new APITokenHandler. The token prefix is
// the env-configured label threaded through to the generator (e.g.
// "sk_live_"); callers should pass cfg.APITokenPrefix.
func NewAPITokenHandler(svc *service.AuthService, prefix string) *APITokenHandler {
	return &APITokenHandler{svc: svc, prefix: prefix}
}

// Mount registers API token routes on the given router. Callers should
// invoke this inside a chi.Route("/sellers/{sellerID}/api-tokens", ...)
// so the sellerID URL param is extracted upstream.
func (h *APITokenHandler) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Issue)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Revoke)
}

// issueAPITokenRequest is the JSON body for creating a new token.
// RateLimitRPS / RateLimitBurst are optional per-token overrides; if
// omitted the gateway falls back to the env-configured default.
type issueAPITokenRequest struct {
	Name           string                  `json:"name"`
	Scopes         []domain.APITokenScope  `json:"scopes"`
	ExpiresAt      *time.Time              `json:"expires_at,omitempty"`
	RateLimitRPS   *int                    `json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int                    `json:"rate_limit_burst,omitempty"`
}

// issueAPITokenResponse embeds the persisted record and adds the plaintext
// token string. The plaintext is returned exactly once — the UI must warn
// the operator to copy it immediately.
type issueAPITokenResponse struct {
	*domain.SellerAPIToken
	Token string `json:"token"`
}

// parseContext extracts tenantID + sellerID from the request. Writes its
// own error response on failure and returns ok=false.
func (h *APITokenHandler) parseContext(w http.ResponseWriter, r *http.Request) (tenantID, sellerID uuid.UUID, ok bool) {
	tid, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return uuid.Nil, uuid.Nil, false
	}
	sidStr := chi.URLParam(r, "sellerID")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller id"})
		return uuid.Nil, uuid.Nil, false
	}
	return tid, sid, true
}

// Issue handles POST /sellers/{sellerID}/api-tokens.
func (h *APITokenHandler) Issue(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	var req issueAPITokenRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	token, plaintext, err := h.svc.IssueAPIToken(
		r.Context(),
		tenantID,
		sellerID,
		req.Name,
		req.Scopes,
		req.RateLimitRPS,
		req.RateLimitBurst,
		req.ExpiresAt,
		h.prefix,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, issueAPITokenResponse{
		SellerAPIToken: token,
		Token:          plaintext,
	})
}

// List handles GET /sellers/{sellerID}/api-tokens.
func (h *APITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	p := pagination.FromRequest(r)
	tokens, total, err := h.svc.ListAPITokens(r.Context(), tenantID, sellerID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	resp := pagination.Response[domain.SellerAPIToken]{
		Items:  tokens,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Get handles GET /sellers/{sellerID}/api-tokens/{id}.
func (h *APITokenHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token id"})
		return
	}
	t, err := h.svc.GetAPIToken(r.Context(), tenantID, sellerID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, t)
}

// revokeAPITokenResponse surfaces the revoked token's (prefix, lookup)
// pair so the caller — typically the gateway's SellerAPITokenHandler —
// can synchronously evict its lookup cache. Without this, a revoked
// token remains usable for up to the gateway cache TTL (30 s) after the
// 200 OK comes back. The lookup is low-entropy and identifies an
// already-revoked row, so echoing it is not a disclosure risk.
type revokeAPITokenResponse struct {
	Status      string `json:"status"`
	TokenPrefix string `json:"token_prefix"`
	TokenLookup string `json:"token_lookup"`
}

// Revoke handles DELETE /sellers/{sellerID}/api-tokens/{id}.
func (h *APITokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token id"})
		return
	}
	prefix, lookup, err := h.svc.RevokeAPIToken(r.Context(), tenantID, sellerID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, revokeAPITokenResponse{
		Status:      "revoked",
		TokenPrefix: prefix,
		TokenLookup: lookup,
	})
}
