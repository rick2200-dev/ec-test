// Package httpclient contains outbound HTTP adapters for the cart service.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/cart/internal/port"
)

// CatalogClient is a small HTTP client for fetching SKU details from the
// catalog service. Used during AddItem to snapshot price/name/seller_id.
// Implements port.SKULookupClient.
type CatalogClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

// NewCatalogClient constructs a CatalogClient pointing at the catalog service.
// internalToken is sent in the X-Internal-Token header on every request and
// must match CATALOG_INTERNAL_TOKEN on the catalog service.
func NewCatalogClient(baseURL, internalToken string) *CatalogClient {
	return &CatalogClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		internalToken: internalToken,
		http:          &http.Client{Timeout: 5 * time.Second},
	}
}

// LookupSKU fetches a SKU by ID from the catalog service.
func (c *CatalogClient) LookupSKU(ctx context.Context, tenantID, skuID uuid.UUID) (*port.SKULookup, error) {
	reqURL := fmt.Sprintf("%s/internal/skus/%s", c.baseURL, skuID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build catalog request: %w", err)
	}
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperrors.Internal("catalog service unreachable", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, apperrors.NotFound(fmt.Sprintf("sku not found: %s", skuID))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.Internal(
			fmt.Sprintf("catalog lookup failed: status=%d body=%s", resp.StatusCode, string(body)),
			errors.New("non-200 from catalog"),
		)
	}

	var sku port.SKULookup
	if err := json.NewDecoder(resp.Body).Decode(&sku); err != nil {
		return nil, apperrors.Internal("decode catalog response", err)
	}
	if sku.Status != "" && sku.Status != "active" {
		return nil, apperrors.BadRequest(fmt.Sprintf("sku is not purchasable (status=%s)", sku.Status))
	}
	return &sku, nil
}
