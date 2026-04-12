// Package httpclient contains outbound HTTP adapters for the order service.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuyerSubscriptionClient is a lightweight HTTP client for querying buyer
// subscription status from the auth service.
// Implements port.BuyerSubscriptionChecker.
type BuyerSubscriptionClient struct {
	baseURL string
	http    *http.Client
}

// NewBuyerSubscriptionClient creates a new client pointing at the auth service.
func NewBuyerSubscriptionClient(authServiceURL string) *BuyerSubscriptionClient {
	return &BuyerSubscriptionClient{
		baseURL: strings.TrimRight(authServiceURL, "/"),
		http: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// buyerSubscriptionResponse is the subset of fields we need from the auth service response.
type buyerSubscriptionResponse struct {
	Status   string                    `json:"status"`
	PlanSlug string                    `json:"plan_slug"`
	Features buyerSubscriptionFeatures `json:"features"`
}

type buyerSubscriptionFeatures struct {
	FreeShipping bool `json:"free_shipping"`
}

// HasFreeShipping checks whether the given buyer has an active subscription
// with free shipping. Returns false if the buyer has no subscription, the
// subscription is inactive, or the auth service is unreachable.
func (c *BuyerSubscriptionClient) HasFreeShipping(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (bool, error) {
	reqURL := c.baseURL + "/buyer-subscriptions/buyers/" + url.PathEscape(buyerAuth0ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var sub buyerSubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}

	return sub.Status == "active" && sub.Features.FreeShipping, nil
}
