package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	gwauthz "github.com/Riku-KANO/ec-test/services/gateway/internal/authz"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// PlatformAdminHandler proxies platform admin management requests from the
// gateway to the auth service. After successful mutations the loader's
// cache for the affected user is evicted so the new role is visible
// immediately within this gateway process.
type PlatformAdminHandler struct {
	auth   *proxy.ServiceClient
	loader *gwauthz.Loader
}

// NewPlatformAdminHandler creates a new PlatformAdminHandler.
func NewPlatformAdminHandler(svc *proxy.Services, loader *gwauthz.Loader) *PlatformAdminHandler {
	return &PlatformAdminHandler{auth: svc.Auth, loader: loader}
}

// List handles GET /admin/admins.
func (h *PlatformAdminHandler) List(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/platform-admins", "")
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Me handles GET /admin/admins/me.
func (h *PlatformAdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/platform-admins/me", "")
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Grant handles POST /admin/admins. Body: {auth0_user_id, role}.
func (h *PlatformAdminHandler) Grant(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	body, status, err := h.auth.Post(r.Context(), "/platform-admins", bytes.NewReader(bodyBytes))
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
			h.loader.EvictPlatformAdminRole(tc.TenantID, payload.Auth0UserID)
		}
	}
	writeRaw(w, status, body)
}

// UpdateRole handles PUT /admin/admins/{id}/role.
func (h *PlatformAdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.auth.Put(r.Context(), "/platform-admins/"+url.PathEscape(id)+"/role", r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Revoke handles DELETE /admin/admins/{id}.
func (h *PlatformAdminHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.auth.Delete(r.Context(), "/platform-admins/"+url.PathEscape(id))
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListAudit handles GET /admin/audit.
func (h *PlatformAdminHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/rbac-audit", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
