package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/domain"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/port"
)

// SellerHandler handles seller-facing shipment routes.
type SellerHandler struct {
	svc port.ShipmentUseCase
}

// NewSellerHandler creates a new SellerHandler.
func NewSellerHandler(svc port.ShipmentUseCase) *SellerHandler {
	return &SellerHandler{svc: svc}
}

// Routes mounts seller shipment routes.
func (h *SellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/register", h.Register)
	r.Post("/{id}/deliver", h.Deliver)
	return r
}

// List handles GET /seller/shipments
func (h *SellerHandler) List(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "seller context required"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	result, err := h.svc.ListBySeller(r.Context(), port.ListShipmentsInput{
		TenantID: tc.TenantID,
		SellerID: *tc.SellerID,
		Status:   r.URL.Query().Get("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		slog.Error("list shipments failed", "error", err)
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"items":  toResponseList(result.Items),
		"total":  result.Total,
		"limit":  result.Limit,
		"offset": result.Offset,
	})
}

// Get handles GET /seller/shipments/{id}
func (h *SellerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "seller context required"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, mapError(domain.ErrShipmentNotFound))
		return
	}

	shipment, err := h.svc.GetByID(r.Context(), tc.TenantID, *tc.SellerID, id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, toResponse(shipment))
}

type registerRequest struct {
	Carrier        string  `json:"carrier"`
	TrackingNumber string  `json:"tracking_number"`
	ShippedAt      *string `json:"shipped_at"`
	Note           string  `json:"note"`
}

// Register handles POST /seller/shipments/{id}/register
func (h *SellerHandler) Register(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "seller context required"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, mapError(domain.ErrShipmentNotFound))
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, mapError(domain.ErrTrackingNumberRequired))
		return
	}

	in := port.RegisterShipmentInput{
		ShipmentID:     id,
		TenantID:       tc.TenantID,
		SellerID:       *tc.SellerID,
		Carrier:        req.Carrier,
		TrackingNumber: req.TrackingNumber,
		Note:           req.Note,
	}
	if req.ShippedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ShippedAt)
		if err != nil {
			httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid shipped_at format"})
			return
		}
		in.ShippedAt = &t
	}

	shipment, err := h.svc.RegisterShipment(r.Context(), in)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, toResponse(shipment))
}

type deliverRequest struct {
	DeliveredAt *string `json:"delivered_at"`
}

// Deliver handles POST /seller/shipments/{id}/deliver
func (h *SellerHandler) Deliver(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "seller context required"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, mapError(domain.ErrShipmentNotFound))
		return
	}

	var req deliverRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	in := port.MarkDeliveredInput{
		ShipmentID: id,
		TenantID:   tc.TenantID,
		SellerID:   *tc.SellerID,
	}
	if req.DeliveredAt != nil {
		t, err := time.Parse(time.RFC3339, *req.DeliveredAt)
		if err != nil {
			httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid delivered_at format"})
			return
		}
		in.DeliveredAt = &t
	}

	shipment, err := h.svc.MarkDelivered(r.Context(), in)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, toResponse(shipment))
}

// SellerOrderShipmentHandler handles GET /seller/orders/{order_id}/shipment
type SellerOrderShipmentHandler struct {
	svc port.ShipmentUseCase
}

// NewSellerOrderShipmentHandler creates a new handler.
func NewSellerOrderShipmentHandler(svc port.ShipmentUseCase) *SellerOrderShipmentHandler {
	return &SellerOrderShipmentHandler{svc: svc}
}

// GetByOrder handles GET /seller/orders/{order_id}/shipment
func (h *SellerOrderShipmentHandler) GetByOrder(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "seller context required"})
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "order_id"))
	if err != nil {
		httputil.Error(w, mapError(domain.ErrShipmentNotFound))
		return
	}

	shipment, err := h.svc.GetByOrderIDSeller(r.Context(), tc.TenantID, *tc.SellerID, orderID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, toResponse(shipment))
}

// BuyerHandler handles buyer-facing shipment routes.
type BuyerHandler struct {
	svc port.ShipmentUseCase
}

// NewBuyerHandler creates a new BuyerHandler.
func NewBuyerHandler(svc port.ShipmentUseCase) *BuyerHandler {
	return &BuyerHandler{svc: svc}
}

// GetByOrder handles GET /buyer/orders/{order_id}/shipment
func (h *BuyerHandler) GetByOrder(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "order_id"))
	if err != nil {
		httputil.Error(w, mapError(domain.ErrShipmentNotFound))
		return
	}

	shipment, err := h.svc.GetByOrderIDBuyer(r.Context(), tc.TenantID, tc.UserID, orderID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, toResponse(shipment))
}

// HealthHandler handles health check routes.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Liveness handles GET /healthz
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness handles GET /readyz
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- response helpers ---

type shipmentResponse struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	SellerID        string  `json:"seller_id"`
	OrderID         string  `json:"order_id"`
	BuyerAuth0ID    string  `json:"buyer_auth0_id"`
	Status          string  `json:"status"`
	Carrier         string  `json:"carrier,omitempty"`
	TrackingNumber  string  `json:"tracking_number,omitempty"`
	Note            string  `json:"note,omitempty"`
	ShippedAt       *string `json:"shipped_at,omitempty"`
	DeliveredAt     *string `json:"delivered_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toResponse(s *domain.Shipment) shipmentResponse {
	resp := shipmentResponse{
		ID:           s.ID.String(),
		TenantID:     s.TenantID.String(),
		SellerID:     s.SellerID.String(),
		OrderID:      s.OrderID.String(),
		BuyerAuth0ID: s.BuyerAuth0ID,
		Status:       s.Status,
		Carrier:      s.Carrier,
		TrackingNumber: s.TrackingNumber,
		Note:         s.Note,
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
	}
	if s.ShippedAt != nil {
		t := s.ShippedAt.Format(time.RFC3339)
		resp.ShippedAt = &t
	}
	if s.DeliveredAt != nil {
		t := s.DeliveredAt.Format(time.RFC3339)
		resp.DeliveredAt = &t
	}
	return resp
}

func toResponseList(items []*domain.Shipment) []shipmentResponse {
	out := make([]shipmentResponse, len(items))
	for i, s := range items {
		out[i] = toResponse(s)
	}
	return out
}

