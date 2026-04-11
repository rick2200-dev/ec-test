package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/middleware/apitoken"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// SellerAPITokenHandler proxies seller API-token management requests
// from the gateway to the auth service. Mirrors SellerTeamHandler's
// shape: seller id is taken from the JWT-derived tenant context rather
// than the URL so sellers can only touch their own tokens.
//
// On successful revoke the handler asks the apitoken.Loader to evict
// its cache entry so the token stops working immediately instead of
// after the 30s TTL. The parallel seller-team handler takes a similar
// approach — see the comment at the top of seller_team_handler.go.
type SellerAPITokenHandler struct {
	auth         *proxy.ServiceClient
	apiTokenLoad *apitoken.Loader
	tokenPrefix  string
}

// NewSellerAPITokenHandler creates a new SellerAPITokenHandler. The
// tokenPrefix must match the auth service's API_TOKEN_PREFIX — it's
// used to key cache eviction on revoke.
func NewSellerAPITokenHandler(svc *proxy.Services, loader *apitoken.Loader, tokenPrefix string) *SellerAPITokenHandler {
	return &SellerAPITokenHandler{
		auth:         svc.Auth,
		apiTokenLoad: loader,
		tokenPrefix:  tokenPrefix,
	}
}

// authPrefix builds the auth-service path prefix for the caller's seller.
func (h *SellerAPITokenHandler) authPrefix(r *http.Request) (string, *uuid.UUID, bool) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.SellerID == nil {
		return "", nil, false
	}
	return "/sellers/" + tc.SellerID.String() + "/api-tokens", tc.SellerID, true
}

// List handles GET /seller/api-tokens.
func (h *SellerAPITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, err := h.auth.Get(r.Context(), prefix, r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Issue handles POST /seller/api-tokens. The auth service returns the
// plaintext token exactly once inside this response — the gateway just
// passes it through unchanged. No cache work is needed because the new
// token has no existing cache entry to evict.
func (h *SellerAPITokenHandler) Issue(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, err := h.auth.Post(r.Context(), prefix, r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Get handles GET /seller/api-tokens/{id}.
func (h *SellerAPITokenHandler) Get(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	id := r.PathValue("id")
	body, status, err := h.auth.Get(r.Context(), prefix+"/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Revoke handles DELETE /seller/api-tokens/{id}. On success, if the
// downstream response carries the token's prefix+lookup, evict them from
// the loader cache so the token cannot keep authenticating within the
// 30s TTL. The auth service returns these fields exactly for this
// purpose (see handler.NewAPITokenHandler's revoke response shape).
func (h *SellerAPITokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	id := r.PathValue("id")
	body, status, err := h.auth.Delete(r.Context(), prefix+"/"+url.PathEscape(id))
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	// Best-effort cache eviction. A successful revoke on the auth side
	// is enough to make the token unusable within the 30s window; this
	// just closes that window to ~immediate.
	if status >= 200 && status < 300 {
		var payload struct {
			TokenPrefix string `json:"token_prefix"`
			TokenLookup string `json:"token_lookup"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.TokenLookup != "" {
			p := payload.TokenPrefix
			if p == "" {
				p = h.tokenPrefix
			}
			h.apiTokenLoad.Evict(p, payload.TokenLookup)
		}
	}
	writeRaw(w, status, body)
}
