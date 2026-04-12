package cancellation

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// HTTPHandler serves the order-cancellation REST endpoints. It is
// mounted in cmd/server/main.go alongside (not nested under) the
// existing OrderHandler routes so this package never has to reach
// into internal/handler.
type HTTPHandler struct {
	svc *Service
}

// NewHTTPHandler constructs an HTTPHandler.
func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// AttachOrderScopedRoutes attaches the buyer-facing cancellation
// endpoints onto a chi router that will be mounted at /orders. This is
// the way to wire the cancellation package's buyer endpoints alongside
// the existing OrderHandler routes, because chi does not allow two
// Mount() calls at the same prefix.
//
// Usage in main.go:
//
//	ordersRouter := orderHandler.Routes()
//	cancellationHandler.AttachOrderScopedRoutes(ordersRouter)
//	r.Mount("/orders", ordersRouter)
//
// which yields:
//
//	POST /orders/{id}/cancellation-request
//	GET  /orders/{id}/cancellation-request
func (h *HTTPHandler) AttachOrderScopedRoutes(r chi.Router) {
	r.Post("/{id}/cancellation-request", h.createRequest)
	r.Get("/{id}/cancellation-request", h.getLatestForOrder)
}

// SellerRoutes returns the chi router for seller-facing cancellation
// endpoints. Intended to be mounted under /cancellation-requests:
//
//	POST /cancellation-requests/{id}/approve
//	POST /cancellation-requests/{id}/reject
//	GET  /cancellation-requests?status=pending&limit=&offset=
//	GET  /cancellation-requests/{id}
func (h *HTTPHandler) SellerRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{id}", h.getByID)
	r.Post("/{id}/approve", h.approve)
	r.Post("/{id}/reject", h.reject)
	return r
}

// --- Request / Response shapes ---

type createRequestBody struct {
	Reason string `json:"reason"`
}

type rejectRequestBody struct {
	SellerComment string `json:"seller_comment"`
}

type approveRequestBody struct {
	SellerComment string `json:"seller_comment"`
}

// --- Buyer handlers ---

func (h *HTTPHandler) createRequest(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	orderID, ok := uuidParam(w, r, "id", "order id")
	if !ok {
		return
	}

	var body createRequestBody
	if err := httputil.Decode(r, &body); err != nil {
		httputil.Error(w, err)
		return
	}
	reason := strings.TrimSpace(body.Reason)

	req, err := h.svc.RequestCancellation(r.Context(), tc.TenantID, orderID, tc.UserID, reason)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, req)
}

func (h *HTTPHandler) getLatestForOrder(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	orderID, ok := uuidParam(w, r, "id", "order id")
	if !ok {
		return
	}

	req, err := h.svc.GetLatestForOrder(r.Context(), tc.TenantID, orderID, tc.UserID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if req == nil {
		// No request has ever been opened against this order. The
		// buyer page uses this to render the "Cancel" button.
		httputil.Error(w, apperrors.NotFound("no cancellation request for this order"))
		return
	}
	httputil.JSON(w, http.StatusOK, req)
}

// --- Seller handlers ---

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.BadRequest("seller context required"))
		return
	}

	statusParam := strings.TrimSpace(r.URL.Query().Get("status"))
	status := StatusPending
	if statusParam != "" {
		status = Status(statusParam)
	}

	p := pagination.FromRequest(r)
	requests, total, err := h.svc.ListByStatus(r.Context(), tc.TenantID, *tc.SellerID, status, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, pagination.Response[CancellationRequest]{
		Items:  requests,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	})
}

func (h *HTTPHandler) getByID(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.BadRequest("seller context required"))
		return
	}
	requestID, ok := uuidParam(w, r, "id", "cancellation request id")
	if !ok {
		return
	}

	// Use the seller-scoped variant so a multi-seller tenant cannot fetch
	// another seller's cancellation reason / buyer identifier via the
	// id-lookup endpoint.
	req, err := h.svc.GetByIDForSeller(r.Context(), tc.TenantID, *tc.SellerID, requestID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, req)
}

func (h *HTTPHandler) approve(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.BadRequest("seller context required"))
		return
	}
	requestID, ok := uuidParam(w, r, "id", "cancellation request id")
	if !ok {
		return
	}

	var body approveRequestBody
	if r.ContentLength > 0 {
		if err := httputil.Decode(r, &body); err != nil {
			httputil.Error(w, err)
			return
		}
	}

	updated, err := h.svc.ApproveCancellation(r.Context(), tc.TenantID, requestID, *tc.SellerID, strings.TrimSpace(body.SellerComment))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, updated)
}

func (h *HTTPHandler) reject(w http.ResponseWriter, r *http.Request) {
	tc, ok := tenantCtx(w, r)
	if !ok {
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.BadRequest("seller context required"))
		return
	}
	requestID, ok := uuidParam(w, r, "id", "cancellation request id")
	if !ok {
		return
	}

	var body rejectRequestBody
	if err := httputil.Decode(r, &body); err != nil {
		httputil.Error(w, err)
		return
	}
	comment := strings.TrimSpace(body.SellerComment)
	if comment == "" {
		httputil.Error(w, apperrors.BadRequest("seller_comment is required when rejecting a cancellation request"))
		return
	}

	updated, err := h.svc.RejectCancellation(r.Context(), tc.TenantID, requestID, *tc.SellerID, comment)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, updated)
}

// --- Small helpers ---

func tenantCtx(w http.ResponseWriter, r *http.Request) (tenant.Context, bool) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return tenant.Context{}, false
	}
	return tc, true
}

func uuidParam(w http.ResponseWriter, r *http.Request, key, label string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid "+label))
		return uuid.Nil, false
	}
	return id, true
}
