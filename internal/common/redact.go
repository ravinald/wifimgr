package common

import (
	"encoding/json"
	"strings"
)

// sensitiveFields are JSON keys whose values must never reach a debug log.
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

// RedactJSON parses data as JSON and replaces the value of any sensitive field
// with "[REDACTED]", recursing through objects and arrays. A body that does not
// parse as JSON is returned as a length placeholder rather than raw bytes — a
// non-JSON error page (HTML, form-encoded) could still carry a token, so the
// fallback fails closed.
func RedactJSON(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "[non-JSON body redacted]"
	}

	result, err := json.Marshal(redactValue(parsed))
	if err != nil {
		return "[redaction failed]"
	}
	return string(result)
}

func redactValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, child := range val {
			if sensitiveFields[strings.ToLower(k)] {
				result[k] = "[REDACTED]"
			} else {
				result[k] = redactValue(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, child := range val {
			result[i] = redactValue(child)
		}
		return result
	default:
		return v
	}
}
