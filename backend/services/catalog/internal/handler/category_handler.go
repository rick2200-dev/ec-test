package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/service"
)

// CategoryHandler handles HTTP requests for category operations.
type CategoryHandler struct {
	svc *service.CatalogService
}

// NewCategoryHandler creates a new CategoryHandler.
func NewCategoryHandler(svc *service.CatalogService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// Routes returns the chi router for category endpoints.
func (h *CategoryHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	return r
}

// createCategoryRequest is the request body for creating a category.
type createCategoryRequest struct {
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	SortOrder int        `json:"sort_order"`
}

// Create handles POST /categories.
func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	var req createCategoryRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	c := &domain.Category{
		ParentID:  req.ParentID,
		Name:      req.Name,
		Slug:      req.Slug,
		SortOrder: req.SortOrder,
	}

	if err := h.svc.CreateCategory(r.Context(), tenantID, c); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, c)
}

// List handles GET /categories.
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	categories, err := h.svc.ListCategories(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, categories)
}

// updateCategoryRequest is the request body for updating a category.
type updateCategoryRequest struct {
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	SortOrder int        `json:"sort_order"`
}

// Update handles PUT /categories/{id}.
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid category id"})
		return
	}

	var req updateCategoryRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	c := &domain.Category{
		ID:        id,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Slug:      req.Slug,
		SortOrder: req.SortOrder,
	}

	if err := h.svc.UpdateCategory(r.Context(), tenantID, c); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, c)
}
