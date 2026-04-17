package ubiquiti

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ravinald/wifimgr/internal/logging"
)

// Client is the HTTP client for the Ubiquiti Site Manager API.
type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	rateLimiter *RateLimiter
	retryConfig *RetryConfig
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client (useful for testing with httptest).
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// NewClient creates a new Ubiquiti Site Manager API client.
func NewClient(apiKey, baseURL string, opts ...ClientOption) *Client {
	if baseURL == "" {
		baseURL = "https://api.ui.com"
	}

	c := &Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		httpClient:  http.DefaultClient,
		rateLimiter: NewRateLimiter(166, 50),
		retryConfig: DefaultRetryConfig(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// doRequest executes an HTTP request against the Site Manager API.
// Handles rate limiting, authentication, and retry on 429.
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values) (*apiResponse, *http.Response, error) {
	retryState := NewRetryState(c.retryConfig)

	for {
		if err := c.rateLimiter.Acquire(ctx); err != nil {
			return nil, nil, fmt.Errorf("rate limit acquire failed: %w", err)
		}

		reqURL := c.baseURL + path
		if len(query) > 0 {
			reqURL += "?" + query.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-API-KEY", c.apiKey)
		req.Header.Set("Accept", "application/json")

		logging.Debugf("[ubiquiti] %s %s", method, path)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, resp, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if retryState.ShouldRetry(resp.StatusCode) {
				if waitErr := retryState.WaitBeforeRetry(ctx, resp); waitErr != nil {
					return nil, resp, fmt.Errorf("retry wait failed: %w", waitErr)
				}
				continue
			}
			return nil, resp, fmt.Errorf("rate limited (429) after %d retries", retryState.Config.MaxRetries)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp, fmt.Errorf("API error: %s (status %d, body: %s)", resp.Status, resp.StatusCode, truncate(string(body), 200))
		}

		logging.Debugf("[ubiquiti] Response body (first 1000 chars): %s", truncate(string(body), 1000))

		var apiResp apiResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, resp, fmt.Errorf("failed to decode response: %w", err)
		}

		return &apiResp, resp, nil
	}
}

// paginate fetches all pages of a paginated API endpoint.
func paginate[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var allItems []T
	query := url.Values{}

	for {
		apiResp, _, err := c.doRequest(ctx, http.MethodGet, path, query)
		if err != nil {
			return nil, err
		}

		// Re-marshal the data field and unmarshal into the target type
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data field: %w", err)
		}

		var items []T
		if err := json.Unmarshal(dataBytes, &items); err != nil {
			return nil, fmt.Errorf("failed to decode data as %T: %w", items, err)
		}

		allItems = append(allItems, items...)

		if apiResp.NextToken == "" {
			break
		}
		query.Set("nextToken", apiResp.NextToken)
	}

	return allItems, nil
}

// GetHosts returns all hosts (controllers/consoles).
func (c *Client) GetHosts(ctx context.Context) ([]Host, error) {
	hosts, err := paginate[Host](ctx, c, "/v1/hosts")
	if err != nil {
		return nil, fmt.Errorf("failed to get hosts: %w", err)
	}
	logging.Debugf("[ubiquiti] Fetched %d hosts", len(hosts))
	return hosts, nil
}

// GetSites returns all sites.
func (c *Client) GetSites(ctx context.Context) ([]Site, error) {
	sites, err := paginate[Site](ctx, c, "/v1/sites")
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}
	logging.Debugf("[ubiquiti] Fetched %d sites", len(sites))
	return sites, nil
}

// GetDevices returns all devices grouped by host.
func (c *Client) GetDevices(ctx context.Context) ([]HostDeviceGroup, error) {
	groups, err := paginate[HostDeviceGroup](ctx, c, "/v1/devices")
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	total := 0
	for _, g := range groups {
		total += len(g.Devices)
	}
	logging.Debugf("[ubiquiti] Fetched %d device groups with %d total devices", len(groups), total)
	return groups, nil
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
