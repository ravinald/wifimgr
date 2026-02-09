package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ravinald/wifimgr/internal/common"
	"github.com/ravinald/wifimgr/internal/logging"
)

// HTTP-related methods for the mistClient

// do executes an HTTP request with the given method, path, and body
// It handles rate limiting, retries, and error handling
func (c *mistClient) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Check if context is already canceled or deadline exceeded
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Build the URL
	url := c.buildURL(path)

	// Prepare request body if provided
	var reqBody io.Reader
	var jsonData []byte
	if body != nil {
		var err error
		jsonData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Log the API request details when debug is enabled
	if c.debug {
		if body != nil && len(jsonData) > 0 {
			// Log with request body for POST/PUT/PATCH
			if len(jsonData) > 1000 {
				// Truncate large payloads
				c.logDebug("API Request: %s %s (body: %d bytes)", method, url, len(jsonData))
			} else {
				c.logDebug("API Request: %s %s (body: %s)", method, url, string(jsonData))
			}
		} else {
			// Log without body for GET/DELETE
			c.logDebug("API Request: %s %s", method, url)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.config.APIToken))

	// Execute the request with retry functionality if enabled
	if c.maxRetries > 0 {
		err = c.retryRequest(ctx, func() (int, error) {
			// Apply rate limiting if configured
			if c.rateLimiter != nil {
				c.rateLimiter.wait()
			}

			// Execute the request
			resp, err := c.httpClient.Do(req)
			if err != nil {
				return 0, err
			}
			defer func() { _ = resp.Body.Close() }()

			// Read the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
			}

			// Handle non-2xx status codes
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return resp.StatusCode, c.handleErrorResponse(resp.StatusCode, bodyBytes)
			}

			// Only parse the result if there is something to parse
			if result != nil && len(bodyBytes) > 0 {
				if err := json.Unmarshal(bodyBytes, result); err != nil {
					return resp.StatusCode, fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}

			return resp.StatusCode, nil
		})
	} else {
		// No retry, just execute the request once
		// Apply rate limiting if configured
		if c.rateLimiter != nil {
			c.rateLimiter.wait()
		}

		// Execute the request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Handle non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return c.handleErrorResponse(resp.StatusCode, bodyBytes)
		}

		// Only parse the result if there is something to parse
		if result != nil && len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}
	}

	return err
}

// handleErrorResponse handles HTTP error responses from the API
func (c *mistClient) handleErrorResponse(statusCode int, body []byte) error {
	// Log the raw error response in debug mode
	if c.debug {
		logging.Debugf("API Error Response [%d]: %s", statusCode, string(body))
	}

	// Try to parse the error response - handle both simple and complex structures
	// Mist API can return:
	// - Simple: {"detail": "error message"}
	// - Array of errors: {"detail": ["error1", "error2"]}
	// - Nested validation: {"detail": {"field": ["error1"]}}
	var rawResp map[string]interface{}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		// If we can't parse at all, return a generic error
		return c.statusCodeToError(statusCode)
	}

	// Extract error message(s) from the response
	errMsg := extractErrorMessages(rawResp)
	if errMsg == "" {
		return c.statusCodeToError(statusCode)
	}

	// Return a detailed error message
	return fmt.Errorf("API error [%d]: %s", statusCode, errMsg)
}

// extractErrorMessages extracts error messages from various API response formats
func extractErrorMessages(resp map[string]interface{}) string {
	// Try common error field names in order of specificity
	for _, field := range []string{"detail", "error", "message", "errors"} {
		if val, ok := resp[field]; ok {
			msg := formatErrorValue(val)
			if msg != "" {
				return msg
			}
		}
	}
	return ""
}

// formatErrorValue formats an error value which can be a string, array, or nested map
func formatErrorValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case []interface{}:
		// Array of errors - join them
		var msgs []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				msgs = append(msgs, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				// Nested error object
				if nested := formatErrorValue(m); nested != "" {
					msgs = append(msgs, nested)
				}
			}
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	case map[string]interface{}:
		// Nested validation errors - format as "field: error"
		var msgs []string
		for field, fieldVal := range v {
			fieldMsg := formatErrorValue(fieldVal)
			if fieldMsg != "" {
				msgs = append(msgs, fmt.Sprintf("%s: %s", field, fieldMsg))
			}
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	}
	return ""
}

// statusCodeToError converts an HTTP status code to a specific error
func (c *mistClient) statusCodeToError(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return errors.New("unauthorized: invalid API token")
	case http.StatusForbidden:
		return errors.New("forbidden: insufficient permissions")
	case http.StatusNotFound:
		return errors.New("not found: resource does not exist")
	case http.StatusTooManyRequests:
		return errors.New("rate limited: too many requests")
	case http.StatusBadRequest:
		return errors.New("bad request: invalid parameters")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("server error: service unavailable (%d)", statusCode)
	default:
		return fmt.Errorf("unexpected status code: %d", statusCode)
	}
}

// buildURL constructs the full API URL from the path
// The base URL should already include /api/v1 (e.g., https://api.ac2.mist.com/api/v1)
func (c *mistClient) buildURL(path string) string {
	// Ensure base URL has a trailing slash
	baseURL := c.config.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Remove any leading slash from the path for consistent handling
	path = strings.TrimPrefix(path, "/")

	// Auto-migrate old configs that don't have /api/v1 in the base URL
	if !strings.Contains(baseURL, "/api/") {
		// Old-style base URL without API version, add it
		c.logDebug("Legacy base URL detected, adding /api/v1/ prefix")
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		baseURL += "api/v1/"
	}

	// Remove any duplicate /api/v1 prefix from the path if it exists
	// This handles the transition period where some code might still include it
	path = strings.TrimPrefix(path, "api/v1/")
	path = strings.TrimPrefix(path, "api/v1")

	// Build the final URL
	finalURL := baseURL + path
	c.logDebug("Built URL: %s", finalURL)
	return finalURL
}

// shouldRetry determines if a request should be retried based on status code and error
func (c *mistClient) shouldRetry(statusCode int, err error) bool {
	// Don't retry if no error and status code is in the 2xx range
	if err == nil && statusCode >= 200 && statusCode < 300 {
		return false
	}

	// Always retry on specific transient errors
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Retry on network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Retry on specific HTTP status codes
	return statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusTooManyRequests
}

// retryRequest executes a function with retry logic
func (c *mistClient) retryRequest(ctx context.Context, fn func() (int, error)) error {
	// If retries are disabled, just execute the function once
	if c.maxRetries <= 0 {
		_, err := fn()
		return err
	}

	var lastErr error
	var lastStatus int

	// Execute with retries
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Check if context is canceled before each attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if attempt > 0 {
			// Only log retries, not the initial attempt
			c.logDebug("Retry attempt %d/%d", attempt, c.maxRetries)
		}

		// Execute the function
		statusCode, err := fn()

		// If successful, return immediately
		if !c.shouldRetry(statusCode, err) {
			return err
		}

		// Store last error and status code for potential return
		lastErr = err
		lastStatus = statusCode

		// Don't sleep after the last attempt
		if attempt < c.maxRetries {
			// Calculate backoff duration with jitter
			backoff := c.calculateBackoff(attempt)

			// For 429 Too Many Requests, try to use the Retry-After header if available
			if statusCode == http.StatusTooManyRequests && err != nil {
				retryAfter := c.extractRetryAfterDuration(err, nil)
				if retryAfter > 0 {
					backoff = retryAfter
				}
			}

			c.logDebug("Backing off for %v before retry", backoff)

			// Use a timer with the context to enable cancellation during sleep
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
				// Timer completed, continue to next iteration
			case <-ctx.Done():
				// Context canceled during sleep
				timer.Stop()
				return ctx.Err()
			}
		}
	}

	// If we get here, all retries failed
	if lastErr != nil {
		return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, lastErr)
	}

	// This is the case where shouldRetry returned true but there was no error
	return fmt.Errorf("request failed after %d attempts with status code %d", c.maxRetries, lastStatus)
}

// calculateBackoff calculates the backoff duration for retries with exponential backoff and jitter
func (c *mistClient) calculateBackoff(attempt int) time.Duration {
	// Start with base backoff duration
	backoff := c.retryBackoff

	// Apply exponential factor (2^attempt)
	if attempt > 0 {
		backoff *= time.Duration(1 << uint(attempt))
	}

	// Add jitter (±20%)
	jitter := int64(float64(backoff.Nanoseconds()) * 0.2 * (rand.Float64()*2 - 1))
	backoff = time.Duration(backoff.Nanoseconds() + jitter)

	// Cap at 60 seconds
	maxBackoff := 60 * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// parseRetryAfter parses the Retry-After header value into seconds
func parseRetryAfter(s string) (int, error) {
	// Try to parse as an integer (seconds)
	if seconds, err := strconv.Atoi(s); err == nil {
		return seconds, nil
	}

	// Try to parse as a HTTP-date
	if t, err := http.ParseTime(s); err == nil {
		// Calculate seconds from now until the specified time
		seconds := int(time.Until(t).Seconds())
		if seconds < 0 {
			// If the time is in the past, use a small delay
			return 1, nil
		}
		return seconds, nil
	}

	// Couldn't parse the value
	return 0, fmt.Errorf("invalid Retry-After value: %s", s)
}

// extractRetryAfterDuration tries to extract the Retry-After duration from an error or response
func (c *mistClient) extractRetryAfterDuration(err error, resp *http.Response) time.Duration {
	// First try to get it from the response if available
	if resp != nil && resp.Header != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try to parse using the helper function
			if seconds, err := parseRetryAfter(retryAfter); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	// Fallback: Check if err.Error() contains a Retry-After hint
	// This is a bit of a hack, but sometimes the error message contains the header information
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Retry-After") {
			// Extract value with a simple regex-like approach
			parts := strings.Split(errMsg, "Retry-After:")
			if len(parts) > 1 {
				valuePart := strings.Split(parts[1], " ")[0]
				valuePart = strings.Trim(valuePart, " \t\n\r;:,")
				if seconds, err := strconv.Atoi(valuePart); err == nil && seconds > 0 {
					return time.Duration(seconds) * time.Second
				}
			}
		}
	}

	return 0
}

// logDebug is defined in client.go

// sensitiveFields lists JSON field names that should be redacted in debug logs
var sensitiveFields = map[string]bool{
	"password":      true,
	"secret":        true,
	"token":         true,
	"api_token":     true,
	"apitoken":      true,
	"api_key":       true,
	"apikey":        true,
	"access_token":  true,
	"refresh_token": true,
	"psk":           true,
	"passphrase":    true,
	"credentials":   true,
	"private_key":   true,
	"auth":          true,
}

// redactSensitiveJSON redacts sensitive fields from JSON content for safe logging
func redactSensitiveJSON(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Try to parse as JSON
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		// Not valid JSON, return as-is (could be form data or plain text)
		return string(data)
	}

	// Recursively redact sensitive fields
	redacted := redactValue(parsed)

	// Re-marshal for logging
	result, err := json.Marshal(redacted)
	if err != nil {
		return "[redaction failed]"
	}
	return string(result)
}

// redactValue recursively redacts sensitive fields from a parsed JSON value
func redactValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			if sensitiveFields[strings.ToLower(k)] {
				result[k] = "[REDACTED]"
			} else {
				result[k] = redactValue(v)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = redactValue(v)
		}
		return result
	default:
		return v
	}
}

// debugTransport is a custom http.RoundTripper that logs HTTP requests and responses
type debugTransport struct {
	transport http.RoundTripper
	client    *mistClient
}

// RoundTrip implements the http.RoundTripper interface
func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.client.debug {
		// Log the request details
		t.client.logDebug("HTTP Request: %s %s", req.Method, req.URL.String())
		t.client.logDebug("Headers:")
		for k, v := range req.Header {
			maskedValue := v
			if k == "Authorization" && len(v) > 0 {
				// Mask token for security
				maskedValue = []string{common.MaskString(v[0])}
			}
			t.client.logDebug("  %s: %s", k, maskedValue)
		}

		// Log request body if present (with sensitive data redacted)
		if req.Body != nil {
			// Read and restore the body
			body, _ := io.ReadAll(req.Body)
			if err := req.Body.Close(); err != nil {
				t.client.logDebug("Warning: failed to close request body: %v", err)
			}

			// Create a new body from the read content
			req.Body = io.NopCloser(bytes.NewBuffer(body))

			// Log the body with sensitive fields redacted
			if len(body) > 0 {
				t.client.logDebug("Request Body:")
				t.client.logDebug("%s", redactSensitiveJSON(body))
			}
		}
	}

	// Execute the request
	start := time.Now()
	resp, err := t.transport.RoundTrip(req)
	duration := time.Since(start)

	if t.client.debug {
		t.client.logDebug("Request duration: %v", duration)

		if err != nil {
			t.client.logDebug("Request error: %v", err)
		} else {
			t.client.logDebug("Response Status: %s", resp.Status)

			// Clone the response body so we can log it without consuming it
			var bodyBytes []byte
			if resp.Body != nil {
				bodyBytes, _ = io.ReadAll(resp.Body)
				if err := resp.Body.Close(); err != nil {
					t.client.logDebug("Warning: failed to close response body: %v", err)
				}
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Log the response body with sensitive data redacted
				if len(bodyBytes) > 0 {
					// For large responses, truncate to a reasonable size
					const maxLogSize = 5000
					bodyToLog := bodyBytes
					if len(bodyBytes) > maxLogSize {
						bodyToLog = bodyBytes[:maxLogSize]
						t.client.logDebug("Response Body (truncated to %d bytes, sensitive fields redacted):", maxLogSize)
					} else {
						t.client.logDebug("Response Body (sensitive fields redacted):")
					}
					t.client.logDebug("%s", redactSensitiveJSON(bodyToLog))
					if len(bodyBytes) > maxLogSize {
						t.client.logDebug("...and %d more bytes", len(bodyBytes)-maxLogSize)
					}
				}
			}
		}

		t.client.logDebug("") // Add an empty line for better readability
	}

	return resp, err
}

// setupDebugTransport configures the debug transport if needed
func setupDebugTransport(c *mistClient) {
	if c.debug && c.httpClient != nil {
		// Only set up the debug transport if not already configured
		_, isDebugTransport := c.httpClient.Transport.(*debugTransport)
		if !isDebugTransport {
			// Use the existing transport or create a default one
			var baseTransport http.RoundTripper
			if c.httpClient.Transport != nil {
				baseTransport = c.httpClient.Transport
			} else {
				baseTransport = http.DefaultTransport
			}

			// Replace with our debug transport
			c.httpClient.Transport = &debugTransport{
				transport: baseTransport,
				client:    c,
			}
		}
	}
}

// rateLimiter provides rate limiting for API requests
type rateLimiter struct {
	limit    int           // Maximum requests per duration
	duration time.Duration // Duration for rate limiting
	tokens   chan struct{} // Token bucket for rate limiting
	stop     chan struct{} // Channel to signal shutdown
}

// newRateLimiter creates a new rate limiter with the specified limit and duration
func newRateLimiter(limit int, duration time.Duration) *rateLimiter {
	if limit <= 0 {
		return nil
	}

	r := &rateLimiter{
		limit:    limit,
		duration: duration,
		tokens:   make(chan struct{}, limit),
		stop:     make(chan struct{}),
	}

	// Fill the token bucket initially
	for i := 0; i < limit; i++ {
		r.tokens <- struct{}{}
	}

	// Start a goroutine to refill tokens periodically
	go func() {
		// Calculate the interval for adding tokens
		interval := duration / time.Duration(limit)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Try to add a token
				select {
				case r.tokens <- struct{}{}:
					// Token added
				default:
					// Bucket is full, skip
				}
			case <-r.stop:
				return
			}
		}
	}()

	return r
}

// wait blocks until a token is available
func (r *rateLimiter) wait() {
	<-r.tokens
}

// Close stops the rate limiter's background goroutine
func (r *rateLimiter) Close() {
	if r == nil {
		return
	}
	select {
	case <-r.stop:
		// Already closed
	default:
		close(r.stop)
	}
}
