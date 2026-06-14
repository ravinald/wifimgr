package aruba

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// IAP REST status codes (envelope "Status-code"), per the Instant REST API Guide.
const (
	statusOK            = 0
	statusInvalidSID    = 1
	statusInvalidAPI    = 2
	statusInvalidJSON   = 3
	statusBadParams     = 4
	statusMissingParams = 5
	statusConfigModule  = 6
	statusInternalComm  = 7
	statusUnknown       = 8
)

// classifyEnvelope maps a decoded IAP response envelope to the wifimgr error
// taxonomy. A 2xx HTTP response can still carry a device-level failure here.
func classifyEnvelope(apiLabel, op string, env *apiEnvelope) error {
	// Some endpoints (login, logout) report success via Status rather than a
	// numeric code; treat a missing code with a Success status as OK.
	code := statusOK
	if env.StatusCode != nil {
		code = *env.StatusCode
	} else if strings.EqualFold(env.Status, "Failed") {
		// Failed with no numeric code (e.g. login failure).
		return &vendors.AuthError{APILabel: apiLabel, Reason: firstNonEmpty(env.ErrorMessage, env.Message, "request failed")}
	}

	msg := firstNonEmpty(env.ErrorMessage, env.Message)

	switch code {
	case statusOK:
		// A "service not enabled" / "master only" notice can arrive as 2xx text.
		if notEnabled := restNotEnabled(env); notEnabled != "" {
			return &restDisabledError{apiLabel: apiLabel, reason: notEnabled}
		}
		return nil
	case statusInvalidSID:
		return &expiredSessionError{apiLabel: apiLabel, reason: firstNonEmpty(msg, "invalid or expired session id")}
	case statusInvalidAPI:
		return &vendors.NotFoundError{APILabel: apiLabel, Resource: firstNonEmpty(msg, op)}
	case statusInvalidJSON, statusBadParams, statusMissingParams:
		return &vendors.TransportError{APILabel: apiLabel, Op: op, Err: fmt.Errorf("status %d: %s", code, firstNonEmpty(msg, "invalid request"))}
	case statusConfigModule:
		return &configModuleError{apiLabel: apiLabel, op: op, reason: firstNonEmpty(msg, "config module error")}
	case statusInternalComm, statusUnknown:
		return &vendors.ServerError{APILabel: apiLabel, Status: code, Err: fmt.Errorf("%s", firstNonEmpty(msg, "device internal error"))}
	default:
		return &vendors.TransportError{APILabel: apiLabel, Op: op, Err: fmt.Errorf("status %d: %s", code, msg)}
	}
}

// classifyHTTP maps a non-2xx HTTP response to the taxonomy. The IAP returns
// 200 for most logical failures, so this mainly catches transport-level faults.
func classifyHTTP(apiLabel, op string, status int, body []byte) error {
	switch {
	case status == 401 || status == 403:
		return &vendors.AuthError{APILabel: apiLabel, Status: status, Reason: truncate(string(body), 200)}
	case status == 404:
		return &vendors.NotFoundError{APILabel: apiLabel, Resource: op}
	case status >= 500:
		return &vendors.ServerError{APILabel: apiLabel, Status: status, Err: fmt.Errorf("%s", truncate(string(body), 200))}
	default:
		return &vendors.TransportError{APILabel: apiLabel, Op: op, Status: status, Err: fmt.Errorf("%s", truncate(string(body), 200))}
	}
}

// restNotEnabled detects the device's "REST API disabled" / "master only"
// notices, which arrive as plain text rather than a numeric error code.
func restNotEnabled(env *apiEnvelope) string {
	hay := strings.ToLower(env.Message + " " + env.ErrorMessage + " " + env.Status + " " + env.CommandOutput)
	switch {
	case strings.Contains(hay, "rest api service is not enabled"):
		return "REST API service is not enabled on the Instant AP"
	case strings.Contains(hay, "available only on the master"):
		return "REST API service is available only on the master Instant AP"
	default:
		return ""
	}
}

// expiredSessionError marks a lapsed session so the client retries after a
// fresh login. Not exported: callers react via isExpiredSession.
type expiredSessionError struct {
	apiLabel string
	reason   string
}

func (e *expiredSessionError) Error() string {
	return fmt.Sprintf("%s: %s", e.apiLabel, e.reason)
}

func isExpiredSession(err error) bool {
	var e *expiredSessionError
	return errors.As(err, &e)
}

// restDisabledError reports that the operator has not run `allow-rest-api`.
// It carries remediation a user can act on immediately.
type restDisabledError struct {
	apiLabel string
	reason   string
}

func (e *restDisabledError) Error() string {
	return fmt.Sprintf("%s: %s", e.apiLabel, e.reason)
}

func (e *restDisabledError) UserMessage() string {
	return fmt.Sprintf(`%s: %s

Enable it on the Instant AP, then retry:
  (InstantAP)(config)# allow-rest-api
  (InstantAP)(config)# end
  (InstantAP)# commit apply`, e.apiLabel, e.reason)
}

// configModuleError wraps a status-6 config failure. A "Profile not found"
// reason on a delete is surfaced as a NotFound for callers that care.
type configModuleError struct {
	apiLabel string
	op       string
	reason   string
}

func (e *configModuleError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.apiLabel, e.op, e.reason)
}

func (e *configModuleError) UserMessage() string {
	return fmt.Sprintf("%s: configuration rejected by the Instant AP: %s", e.apiLabel, e.reason)
}

// truncate shortens a string for inclusion in error messages.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
