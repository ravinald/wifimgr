package api

import (
	"errors"
	"fmt"
)

// HTTP-status sentinel errors. Callers use errors.Is to classify failures
// without parsing message strings.
var (
	ErrUnauthorized = errors.New("api: unauthorized — invalid API token")
	ErrForbidden    = errors.New("api: forbidden — insufficient permissions")
	ErrNotFound     = errors.New("api: not found — resource does not exist")
	ErrRateLimited  = errors.New("api: rate limited — too many requests")
	ErrBadRequest   = errors.New("api: bad request — invalid parameters")
)

// APIError carries a parsed error response from the upstream API.
// Callers can errors.As(err, &apiErr) to read StatusCode and Message.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api: status %d: %s", e.StatusCode, e.Message)
}
