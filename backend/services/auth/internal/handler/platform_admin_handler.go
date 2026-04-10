package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
)

// PlatformAdminHandler handles HTTP requests for platform admin
// (platform_admins) management and RBAC audit log retrieval.
type PlatformAdminHandler struct {
	svc *service.AuthService
}

// NewPlatformAdminHandler creates a new PlatformAdminHandler.
func NewPlatformAdminHandler(svc *service.AuthService) *PlatformAdminHandler {
	return &PlatformAdminHandler{svc: svc}
}

// Routes returns the chi router for platform admin endpoints.
func (h *PlatformAdminHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Grant)
	r.Get("/me", h.Me)
	r.Put("/{id}/role", h.UpdateRole)
	r.Delete("/{id}", h.Revoke)
	return r
}

// AuditRoutes returns the chi router for the RBAC audit log.
func (h *PlatformAdminHandler) AuditRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListAudit)
	return r
}

type grantPlatformAdminRequest struct {
	Auth0UserID string                   `json:"auth0_user_id"`
	Role        domain.PlatformAdminRole `json:"role"`
}

type updatePlatformAdminRoleRequest struct {
	Role domain.PlatformAdminRole `json:"role"`
}

// List handles GET /platform-admins.
func (h *PlatformAdminHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	admins, err := h.svc.ListPlatformAdmins(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, admins)
}

// Grant handles POST /platform-admins.
func (h *PlatformAdminHandler) Grant(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	var req grantPlatformAdminRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	created, err := h.svc.GrantPlatformAdmin(r.Context(), tenantID, req.Auth0UserID, req.Role)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, created)
}

// UpdateRole handles PUT /platform-admins/{id}/role.
func (h *PlatformAdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid admin id"})
		return
	}
	var req updatePlatformAdminRoleRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.svc.UpdatePlatformAdminRole(r.Context(), tenantID, id, req.Role); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Revoke handles DELETE /platform-admins/{id}.
func (h *PlatformAdminHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid admin id"})
		return
	}
	if err := h.svc.RevokePlatformAdmin(r.Context(), tenantID, id); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// Me handles GET /platform-admins/me. Returns the caller's platform admin
// role so UI can conditionally render controls.
func (h *PlatformAdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.UserID == "" {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "caller identity required"})
		return
	}
	role, err := h.svc.LookupPlatformAdminRole(r.Context(), tenantID, tc.UserID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, meResponse{Auth0UserID: tc.UserID, Role: string(role)})
}

// ListAudit handles GET /platform-admins/audit.
func (h *PlatformAdminHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}
	p := pagination.FromRequest(r)
	entries, total, err := h.svc.ListRBACAuditLog(r.Context(), tenantID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, pagination.Response[domain.RBACAuditEntry]{
		Items:  entries,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	})
}
