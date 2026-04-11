package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/Riku-KANO/ec-test/gen/go/catalog/v1"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// productJSON mirrors the shape of domain.Product that the catalog HTTP
// handler previously returned to the buyer frontend, so migrating the
// underlying transport to gRPC is transparent to clients.
//
// Kept in the gateway package (instead of importing catalog's internal
// domain package) because services in this repo do not share internal
// packages across service boundaries.
type productJSON struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	SellerID    string          `json:"seller_id"`
	CategoryID  *string         `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// skuJSON mirrors the REST shape of a SKU belonging to a product. Note
// that the REST shape splits price into `price_amount` + `price_currency`
// whereas the proto uses a nested `Money` message, so this converter
// flattens the price on the way out.
type skuJSON struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	ProductID     string          `json:"product_id"`
	SellerID      string          `json:"seller_id"`
	SKUCode       string          `json:"sku_code"`
	PriceAmount   int64           `json:"price_amount"`
	PriceCurrency string          `json:"price_currency"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	Status        string          `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// productWithSKUsJSON matches the catalog REST `GetProduct` response.
// Uses struct embedding so the top-level fields serialize flat.
type productWithSKUsJSON struct {
	productJSON
	SKUs []skuJSON `json:"skus"`
}

// protoProductToJSON converts a proto Product message into the REST JSON
// shape used by ListProducts. SKUs are intentionally dropped — the REST
// list endpoint never included them.
func protoProductToJSON(p *catalogv1.Product) productJSON {
	if p == nil {
		return productJSON{}
	}
	return productJSON{
		ID:          p.GetId(),
		TenantID:    p.GetTenantId(),
		SellerID:    p.GetSellerId(),
		CategoryID:  nil, // proto has no category_id field
		Name:        p.GetName(),
		Slug:        p.GetSlug(),
		Description: p.GetDescription(),
		Status:      p.GetStatus(),
		Attributes:  attrsToRawJSON(p.GetAttributesJson()),
		CreatedAt:   p.GetCreatedAt().AsTime(),
		UpdatedAt:   p.GetUpdatedAt().AsTime(),
	}
}

// protoProductWithSKUsToJSON converts a proto Product (with nested SKUs)
// to the REST shape used by GetProduct.
func protoProductWithSKUsToJSON(p *catalogv1.Product) productWithSKUsJSON {
	base := protoProductToJSON(p)
	out := productWithSKUsJSON{productJSON: base, SKUs: []skuJSON{}}
	if p == nil {
		return out
	}
	for _, s := range p.GetSkus() {
		out.SKUs = append(out.SKUs, skuJSON{
			ID:            s.GetId(),
			TenantID:      s.GetTenantId(),
			ProductID:     s.GetProductId(),
			SellerID:      s.GetSellerId(),
			SKUCode:       s.GetSkuCode(),
			PriceAmount:   s.GetPrice().GetAmount(),
			PriceCurrency: s.GetPrice().GetCurrency(),
			Attributes:    attrsToRawJSON(s.GetAttributesJson()),
			Status:        s.GetStatus(),
			CreatedAt:     s.GetCreatedAt().AsTime(),
			UpdatedAt:     s.GetUpdatedAt().AsTime(),
		})
	}
	return out
}

// attrsToRawJSON parses a stringified JSON attribute blob from the proto
// into a RawMessage. Invalid or empty inputs are dropped so the field
// disappears via omitempty instead of confusing clients with `null` or
// escaped strings.
func attrsToRawJSON(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	var probe any
	if err := json.Unmarshal([]byte(s), &probe); err != nil {
		return nil
	}
	return json.RawMessage(s)
}

// writeGRPCError maps a gRPC error to an appropriate HTTP response.
// Pilot uses this for the catalog buyer read path; extend when more
// handlers migrate to gRPC.
func writeGRPCError(w http.ResponseWriter, op string, err error) {
	st, ok := status.FromError(err)
	if !ok {
		slog.Error("gRPC call failed (non-status error)", "op", op, "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "upstream service unavailable"})
		return
	}

	slog.Error("gRPC call failed", "op", op, "code", st.Code().String(), "message", st.Message())

	switch st.Code() {
	case codes.NotFound:
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": st.Message()})
	case codes.InvalidArgument:
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": st.Message()})
	case codes.PermissionDenied:
		httputil.JSON(w, http.StatusForbidden, map[string]string{"error": st.Message()})
	case codes.Unauthenticated:
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": st.Message()})
	case codes.AlreadyExists:
		httputil.JSON(w, http.StatusConflict, map[string]string{"error": st.Message()})
	case codes.DeadlineExceeded, codes.Unavailable:
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "upstream service unavailable"})
	default:
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "upstream error"})
	}
}
