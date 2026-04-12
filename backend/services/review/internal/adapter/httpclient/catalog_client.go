// Package httpclient contains outbound HTTP adapters for the review service.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
	"github.com/Riku-KANO/ec-test/services/review/internal/port"
)

// CatalogClient talks to the catalog service's internal product lookup
// endpoint so the review service can resolve product metadata and SKU IDs
// for purchase verification.
// Implements port.CatalogClient.
type CatalogClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

// NewCatalogClient constructs a CatalogClient pointing at the catalog service.
func NewCatalogClient(baseURL, internalToken string) *CatalogClient {
	return &CatalogClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		internalToken: internalToken,
		http:          &http.Client{Timeout: 5 * time.Second},
	}
}

// GetProduct retrieves product metadata and SKU IDs from the catalog service.
func (c *CatalogClient) GetProduct(
	ctx context.Context,
	tenantID, productID uuid.UUID,
) (*port.ProductLookup, error) {
	reqURL := fmt.Sprintf("%s/internal/products/%s", c.baseURL, productID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, apperrors.Internal("build catalog request", err)
	}
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperrors.Internal("catalog service unreachable", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrProductNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, apperrors.Internal(
			fmt.Sprintf("catalog lookup failed: status=%d body=%s", resp.StatusCode, string(raw)),
			nil,
		)
	}

	var result port.ProductLookup
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, apperrors.Internal("decode catalog response", err)
	}
	return &result, nil
}
