package httpclient

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
	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
)

// OrderClient calls the order service's internal checkout endpoint to
// materialize a cart into orders + a PaymentIntent.
// Implements port.CheckoutClient.
type OrderClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

// NewOrderClient constructs an OrderClient pointing at the order service.
// internalToken is sent in the X-Internal-Token header on every request
// and must match ORDER_INTERNAL_TOKEN on the order service.
func NewOrderClient(baseURL, internalToken string) *OrderClient {
	return &OrderClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		internalToken: internalToken,
		// Checkout fans out across multiple DB inserts + a Stripe call.
		// Give it a generous timeout to absorb Stripe latency spikes.
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateCheckout posts the cart's contents to POST /internal/checkouts
// and returns the result.
func (c *OrderClient) CreateCheckout(ctx context.Context, tenantID uuid.UUID, in domain.CheckoutInput) (*domain.CheckoutResult, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, apperrors.Internal("marshal checkout input", err)
	}

	reqURL := c.baseURL + "/internal/checkouts"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, apperrors.Internal("build checkout request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperrors.Internal("order service unreachable", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			if resp.StatusCode == http.StatusBadRequest {
				return nil, apperrors.BadRequest(errResp.Error)
			}
			return nil, apperrors.Internal(errResp.Error, nil)
		}
		return nil, apperrors.Internal(
			fmt.Sprintf("order checkout failed: status=%d body=%s", resp.StatusCode, string(body)),
			nil,
		)
	}

	var result domain.CheckoutResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, apperrors.Internal("decode checkout response", err)
	}
	return &result, nil
}
