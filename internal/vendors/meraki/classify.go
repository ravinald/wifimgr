package meraki

import (
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// ClassifyError converts a (*resty.Response, error) pair returned by the
// Meraki SDK into a wifimgr-typed error from internal/vendors.
//
// Rules:
//   - (nil, nil)          → nil (no error)
//   - (resp, nil)         → nil if 2xx, else classified by status code
//   - (nil, err)          → TransportError{Retryable: true, Status: 0} (network failure before a response)
//   - (resp, err)         → classified by resp.StatusCode(); err preserved in Unwrap chain
//
// The returned error is suitable for errors.As against any of the taxonomy
// types. Callers in retry loops should check via errors.As; they no longer
// need to string-match on the err message.
func ClassifyError(apiLabel, op string, resp *resty.Response, err error) error {
	if err == nil && resp == nil {
		return nil
	}

	// Network or pre-response failure — we never saw a status code. Treat as
	// retryable transport: connection resets, DNS blips, TLS handshake
	// timeouts all fall here and are worth another attempt.
	if resp == nil {
		return &vendors.TransportError{
			APILabel:  apiLabel,
			Op:        op,
			Status:    0,
			Retryable: true,
			Err:       err,
		}
	}

	status := resp.StatusCode()

	// Success per Meraki convention.
	if status >= 200 && status < 300 {
		return nil
	}

	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		reason := "credentials rejected"
		if status == http.StatusForbidden {
			reason = "operation not permitted for this token"
		}
		return &vendors.AuthError{APILabel: apiLabel, Status: status, Reason: reason}

	case http.StatusTooManyRequests:
		return &vendors.RateLimitError{
			APILabel:   apiLabel,
			RetryAfter: ParseRetryAfter(resp.RawResponse),
		}

	case http.StatusNotFound:
		// Transport 404. The resource identifier isn't easy to recover from
		// resty without the caller's context, so leave Resource blank; the
		// caller decorates if it cares.
		return &vendors.NotFoundError{APILabel: apiLabel, Resource: op}
	}

	// 5xx.
	if status >= 500 && status < 600 {
		return &vendors.ServerError{APILabel: apiLabel, Status: status, Err: err}
	}

	// Other 4xx (400, 422, 409, …): non-retryable client errors. Wrap as a
	// non-retryable TransportError so the retry loop gives up.
	return &vendors.TransportError{
		APILabel:  apiLabel,
		Op:        op,
		Status:    status,
		Retryable: false,
		Err:       err,
	}
}
