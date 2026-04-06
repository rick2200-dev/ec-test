package handler

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// AdminHandler handles platform admin routes.
type AdminHandler struct {
	// TODO: add service clients for auth, catalog, etc.
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// ListTenants lists all tenants.
// GET /tenants
func (h *AdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"tenants": []any{},
		"message": "stub: proxy to auth service",
	})
}

// CreateTenant creates a new tenant.
// POST /tenants
func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"message": "stub: proxy to auth service",
	})
}

// ListSellers lists all sellers.
// GET /sellers
func (h *AdminHandler) ListSellers(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"sellers": []any{},
		"message": "stub: proxy to auth service",
	})
}

// ApproveSeller approves a seller.
// PUT /sellers/{id}/approve
func (h *AdminHandler) ApproveSeller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "stub: proxy to auth service",
	})
}
