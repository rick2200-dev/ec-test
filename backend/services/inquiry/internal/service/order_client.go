package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
)

// OrderClient talks to the order service's internal purchase-check endpoint
// so the inquiry service can verify a buyer actually bought the SKU they
// want to contact the seller about.
type OrderClient struct {
	baseURL string
	http    *http.Client
}

// NewOrderClient constructs an OrderClient pointing at the order service.
func NewOrderClient(baseURL string) *OrderClient {
	return &OrderClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		// Purchase check is a single indexed read; a tight timeout keeps a
		// slow order service from blocking inquiry creation.
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

// PurchaseCheckResult mirrors the order service's internal purchase-check
// response shape.
type PurchaseCheckResult struct {
	Purchased       bool      `json:"purchased"`
	EarliestOrderID uuid.UUID `json:"earliest_order_id,omitempty"`
	ProductName     string    `json:"product_name,omitempty"`
	SKUCode         string    `json:"sku_code,omitempty"`
}

// CheckPurchase asks the order service whether the given buyer has a
// paid-or-later order containing skuID from the given seller. The response
// also carries the product_name / sku_code snapshot from the order line,
// which the inquiry service uses to seed a new thread without a catalog
// lookup.
func (c *OrderClient) CheckPurchase(
	ctx context.Context,
	tenantID uuid.UUID,
	buyerAuth0ID string,
	sellerID, skuID uuid.UUID,
) (*PurchaseCheckResult, error) {
	body := map[string]any{
		"buyer_auth0_id": buyerAuth0ID,
		"seller_id":      sellerID,
		"sku_id":         skuID,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, apperrors.Internal("marshal purchase check", err)
	}

	reqURL := c.baseURL + "/internal/purchase-check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, apperrors.Internal("build purchase check request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-User-ID", buyerAuth0ID)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperrors.Internal("order service unreachable", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(raw, &errResp); jsonErr == nil && errResp.Error != "" {
			if resp.StatusCode == http.StatusBadRequest {
				return nil, apperrors.BadRequest(errResp.Error)
			}
			return nil, apperrors.Internal(errResp.Error, nil)
		}
		return nil, apperrors.Internal(
			fmt.Sprintf("purchase check failed: status=%d body=%s", resp.StatusCode, string(raw)),
			nil,
		)
	}

	var result PurchaseCheckResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, apperrors.Internal("decode purchase check response", err)
	}
	return &result, nil
}
