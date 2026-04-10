package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
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
