// Package httpclient contains outbound HTTP adapters for the order service.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// SellerClient is an HTTP client for the auth service's internal seller
// batch-lookup endpoint. Used by the order service at checkout time to
// snapshot seller_name without reading auth_svc directly. Replaces the
// earlier cross-schema JOIN that order performed in its own transaction.
//
// Implements port.SellerLookup.
type SellerClient struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

// NewSellerClient constructs a client pointing at the auth service's base
// URL (e.g. http://auth:8080). internalToken is sent as X-Internal-Token
// header; the auth handler rejects calls without it.
func NewSellerClient(baseURL, internalToken string) *SellerClient {
	return &SellerClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		http:          &http.Client{Timeout: 5 * time.Second},
	}
}

type batchGetSellersRequest struct {
	IDs []string `json:"ids"`
}

type batchGetSellersResponse struct {
	Sellers []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"sellers"`
}

// BatchGetSellerNames returns a map seller_id -> name. Unknown ids are
// silently omitted.
func (c *SellerClient) BatchGetSellerNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	rawIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		rawIDs = append(rawIDs, id.String())
	}
	body, err := json.Marshal(batchGetSellersRequest{IDs: rawIDs})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/sellers/batch-get", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth batch-get sellers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth batch-get sellers returned %s", resp.Status)
	}

	var decoded batchGetSellersResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out := make(map[uuid.UUID]string, len(decoded.Sellers))
	for _, s := range decoded.Sellers {
		id, err := uuid.Parse(s.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q in response: %w", s.ID, err)
		}
		out[id] = s.Name
	}
	return out, nil
}
