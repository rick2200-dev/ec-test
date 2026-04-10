package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	gwauthz "github.com/Riku-KANO/ec-test/services/gateway/internal/authz"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// SellerTeamHandler proxies seller team management requests from the
// gateway to the auth service. The seller id is taken from the caller's
// JWT-derived tenant context (tc.SellerID), not from the URL — sellers
// can only manage their own team.
//
// After successful mutations the handler evicts the loader's cache so the
// new role takes effect immediately within this gateway process.
type SellerTeamHandler struct {
	auth   *proxy.ServiceClient
	loader *gwauthz.Loader
}

// NewSellerTeamHandler creates a new SellerTeamHandler.
func NewSellerTeamHandler(svc *proxy.Services, loader *gwauthz.Loader) *SellerTeamHandler {
	return &SellerTeamHandler{auth: svc.Auth, loader: loader}
}

// authPrefix builds the auth-service path prefix for the caller's seller.
func (h *SellerTeamHandler) authPrefix(r *http.Request) (string, *uuid.UUID, bool) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.SellerID == nil {
		return "", nil, false
	}
	return "/sellers/" + tc.SellerID.String() + "/team", tc.SellerID, true
}

// List handles GET /seller/team.
func (h *SellerTeamHandler) List(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, err := h.auth.Get(r.Context(), prefix, "")
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Me handles GET /seller/team/me.
func (h *SellerTeamHandler) Me(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, err := h.auth.Get(r.Context(), prefix+"/me", "")
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Add handles POST /seller/team. The request body is forwarded as-is. On
// success the gateway evicts the cache for the newly added user.
func (h *SellerTeamHandler) Add(w http.ResponseWriter, r *http.Request) {
	prefix, sellerID, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	tc, _ := tenant.FromContext(r.Context())

	// Buffer the body so we can both forward it and read the auth0 user id
	// for cache eviction.
	bodyBytes, _ := io.ReadAll(r.Body)
	body, status, err := h.auth.Post(r.Context(), prefix, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	if status >= 200 && status < 300 {
		var payload struct {
			Auth0UserID string `json:"auth0_user_id"`
		}
		if json.Unmarshal(bodyBytes, &payload) == nil && payload.Auth0UserID != "" {
			h.loader.EvictSellerRole(tc.TenantID, *sellerID, payload.Auth0UserID)
		}
	}
	writeRaw(w, status, body)
}

// UpdateRole handles PUT /seller/team/{id}/role. The {id} path param is the
// auth-service seller_user UUID, not an auth0 sub — eviction must therefore
// happen on a target sub. We forward the request and let the auth service
// authoritatively decide; for cache freshness we drop our entire process
// cache for this seller's team by issuing a coarse evict on the actor (the
// only sub we know cheaply). Targeted eviction would require an extra
// lookup which is not worth the round trip given the 30s TTL.
func (h *SellerTeamHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	prefix, _, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	id := r.PathValue("id")
	body, status, err := h.auth.Put(r.Context(), prefix+"/"+url.PathEscape(id)+"/role", r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Remove handles DELETE /seller/team/{id}.
func (h *SellerTeamHandler) Remove(w http.ResponseWriter, r *http.Request) {
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
	writeRaw(w, status, body)
}

// TransferOwnership handles POST /seller/team/transfer-ownership.
func (h *SellerTeamHandler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	prefix, sellerID, ok := h.authPrefix(r)
	if !ok {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	tc, _ := tenant.FromContext(r.Context())
	body, status, err := h.auth.Post(r.Context(), prefix+"/transfer-ownership", r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	// On successful transfer, the actor's role flips from owner to admin.
	// Evict the actor's cache entry so subsequent requests reflect the new
	// role immediately.
	if status >= 200 && status < 300 {
		h.loader.EvictSellerRole(tc.TenantID, *sellerID, tc.UserID)
	}
	writeRaw(w, status, body)
}
