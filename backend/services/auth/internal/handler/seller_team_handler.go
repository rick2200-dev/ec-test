package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
)

// SellerTeamHandler handles HTTP requests for seller team (seller_users)
// management. Mounted at /sellers/{sellerID}/team.
type SellerTeamHandler struct {
	svc *service.AuthService
}

// NewSellerTeamHandler creates a new SellerTeamHandler.
func NewSellerTeamHandler(svc *service.AuthService) *SellerTeamHandler {
	return &SellerTeamHandler{svc: svc}
}

// Mount registers seller team routes on the given router. Callers should
// invoke this inside a chi.Route("/sellers/{sellerID}/team", ...) so the
// sellerID URL param is extracted upstream.
func (h *SellerTeamHandler) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Add)
	r.Get("/me", h.Me)
	r.Post("/transfer-ownership", h.TransferOwnership)
	r.Put("/{id}/role", h.UpdateRole)
	r.Delete("/{id}", h.Remove)
}

type addSellerUserRequest struct {
	Auth0UserID string                `json:"auth0_user_id"`
	Role        domain.SellerUserRole `json:"role"`
}

type updateSellerUserRoleRequest struct {
	Role domain.SellerUserRole `json:"role"`
}

type transferOwnershipRequest struct {
	NewOwnerID uuid.UUID `json:"new_owner_id"`
}

type meResponse struct {
	Auth0UserID string `json:"auth0_user_id"`
	Role        string `json:"role"`
}

// parseContext extracts tenant ID, seller ID, and the actor's Auth0 user ID
// from the request context and the sellerID URL parameter. Returns a written
// response and false on failure.
func (h *SellerTeamHandler) parseContext(w http.ResponseWriter, r *http.Request) (tenantID, sellerID uuid.UUID, ok bool) {
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

// List handles GET /sellers/{sellerID}/team.
func (h *SellerTeamHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	users, err := h.svc.ListSellerTeam(r.Context(), tenantID, sellerID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, users)
}

// Add handles POST /sellers/{sellerID}/team.
func (h *SellerTeamHandler) Add(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	var req addSellerUserRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	created, err := h.svc.AddSellerUser(r.Context(), tenantID, sellerID, req.Auth0UserID, req.Role)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, created)
}

// UpdateRole handles PUT /sellers/{sellerID}/team/{id}/role.
func (h *SellerTeamHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	var req updateSellerUserRoleRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.svc.UpdateSellerUserRole(r.Context(), tenantID, sellerID, id, req.Role); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Remove handles DELETE /sellers/{sellerID}/team/{id}.
func (h *SellerTeamHandler) Remove(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	if err := h.svc.RemoveSellerUser(r.Context(), tenantID, sellerID, id); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// TransferOwnership handles POST /sellers/{sellerID}/team/transfer-ownership.
func (h *SellerTeamHandler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	var req transferOwnershipRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.svc.TransferSellerOwnership(r.Context(), tenantID, sellerID, req.NewOwnerID); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "transferred"})
}

// Me handles GET /sellers/{sellerID}/team/me. Returns the caller's role in
// the seller organization so the UI can conditionally render controls.
func (h *SellerTeamHandler) Me(w http.ResponseWriter, r *http.Request) {
	tenantID, sellerID, ok := h.parseContext(w, r)
	if !ok {
		return
	}
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.UserID == "" {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "caller identity required"})
		return
	}
	role, err := h.svc.LookupSellerRole(r.Context(), tenantID, sellerID, tc.UserID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, meResponse{Auth0UserID: tc.UserID, Role: string(role)})
}
