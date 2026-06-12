package common

import "testing"

func TestRedactJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"redacts token", `{"api_token":"nbt_abc.def"}`, `{"api_token":"[REDACTED]"}`},
		{"redacts nested psk", `{"wlan":{"auth":{"psk":"hunter2"}}}`, `{"wlan":{"auth":"[REDACTED]"}}`},
		{"redacts inside array", `[{"secret":"x"}]`, `[{"secret":"[REDACTED]"}]`},
		{"keeps non-sensitive", `{"name":"AP-1"}`, `{"name":"AP-1"}`},
		// A non-JSON body could be an HTML error page hiding a token; fail closed.
		{"non-json fails closed", `<html>token=nbt_abc</html>`, "[non-JSON body redacted]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactJSON([]byte(tt.in)); got != tt.want {
				t.Errorf("RedactJSON(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
