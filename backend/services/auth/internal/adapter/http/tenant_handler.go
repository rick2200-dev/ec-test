package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// TenantHandler handles HTTP requests for tenant operations.
type TenantHandler struct {
	svc port.AuthUseCase
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(svc port.AuthUseCase) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// Routes returns the chi router for tenant endpoints.
func (h *TenantHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	return r
}

// createTenantRequest is the request body for creating a tenant.
type createTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Create handles POST /tenants.
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTenantRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	t := &domain.Tenant{
		Name: req.Name,
		Slug: req.Slug,
	}

	if err := h.svc.CreateTenant(r.Context(), t); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, t)
}

// GetByID handles GET /tenants/{id}.
func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant id"})
		return
	}

	t, err := h.svc.GetTenant(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, t)
}

// List handles GET /tenants.
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)

	tenants, total, err := h.svc.ListTenants(r.Context(), p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	resp := pagination.Response[domain.Tenant]{
		Items:  tenants,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}
