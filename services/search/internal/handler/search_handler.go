package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
	"github.com/Riku-KANO/ec-test/services/search/internal/service"
)

// SearchHandler handles HTTP requests for product search.
type SearchHandler struct {
	svc *service.SearchService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Routes returns the chi router for search endpoints.
func (h *SearchHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Search)
	r.Get("/suggest", h.Suggest)
	return r
}

// Search handles GET /search?q=...&category_id=...&min_price=...&max_price=...&sort=...&limit=...&offset=...
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseSearchRequest(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.svc.Search(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, result)
}

// Suggest handles GET /search/suggest?q=...
func (h *SearchHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseSearchRequest(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.svc.Suggest(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, result)
}

func (h *SearchHandler) parseSearchRequest(r *http.Request) (domain.SearchRequest, error) {
	q := r.URL.Query()

	req := domain.SearchRequest{
		Query: q.Get("q"),
	}

	// Tenant ID from context (set by middleware) or query param
	if tid, err := tenant.TenantID(r.Context()); err == nil {
		req.TenantID = tid
	} else if tidStr := q.Get("tenant_id"); tidStr != "" {
		parsed, err := uuid.Parse(tidStr)
		if err != nil {
			return req, err
		}
		req.TenantID = parsed
	}

	if sellerIDStr := q.Get("seller_id"); sellerIDStr != "" {
		sid, err := uuid.Parse(sellerIDStr)
		if err == nil {
			req.SellerID = &sid
		}
	}

	if catIDStr := q.Get("category_id"); catIDStr != "" {
		cid, err := uuid.Parse(catIDStr)
		if err == nil {
			req.CategoryID = &cid
		}
	}

	if minStr := q.Get("min_price"); minStr != "" {
		if v, err := strconv.ParseFloat(minStr, 64); err == nil {
			req.MinPrice = &v
		}
	}

	if maxStr := q.Get("max_price"); maxStr != "" {
		if v, err := strconv.ParseFloat(maxStr, 64); err == nil {
			req.MaxPrice = &v
		}
	}

	req.Status = q.Get("status")
	req.SortBy = q.Get("sort")
	req.SortOrder = q.Get("order")

	if limitStr := q.Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = v
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = v
		}
	}

	return req, nil
}
