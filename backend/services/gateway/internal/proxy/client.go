package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// ServiceClient is a reusable HTTP client that proxies requests to a
// downstream micro-service, forwarding tenant context headers.
type ServiceClient struct {
	baseURL string
	http    *http.Client
}

// NewServiceClient creates a ServiceClient for the given base URL.
func NewServiceClient(baseURL string) *ServiceClient {
	return &ServiceClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get sends a GET request to the downstream service.
// query is the raw query string (may be empty).
func (c *ServiceClient) Get(ctx context.Context, path string, query string) ([]byte, int, error) {
	url := c.baseURL + path
	if query != "" {
		url += "?" + query
	}
	return c.do(ctx, http.MethodGet, url, nil)
}

// Post sends a POST request with the given body to the downstream service.
func (c *ServiceClient) Post(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	url := c.baseURL + path
	return c.do(ctx, http.MethodPost, url, body)
}

// Put sends a PUT request with the given body to the downstream service.
func (c *ServiceClient) Put(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	url := c.baseURL + path
	return c.do(ctx, http.MethodPut, url, body)
}

// do executes the HTTP request, attaching tenant context headers.
func (c *ServiceClient) do(ctx context.Context, method, url string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy: build request: %w", err)
	}

	// Forward tenant context headers.
	if tc, err := tenant.FromContext(ctx); err == nil {
		req.Header.Set("X-Tenant-ID", tc.TenantID.String())
		req.Header.Set("X-User-ID", tc.UserID)
		if tc.SellerID != nil {
			req.Header.Set("X-Seller-ID", tc.SellerID.String())
		}
		if len(tc.Roles) > 0 {
			req.Header.Set("X-Roles", strings.Join(tc.Roles, ","))
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("proxy: read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
