package meraki

import (
	"errors"
	"testing"
)

func TestIsNonWirelessNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "meraki wireless-only endpoint refusal matches",
			err: errors.New(`error with operation: https://api.meraki.com/api/v1/networks/N_xxx/wireless/clients/connectionStats Error:
 {"errors":["This endpoint only supports wireless networks"]}`),
			want: true,
		},
		{
			name: "case variations still match",
			err:  errors.New(`{"errors":["this endpoint ONLY supports wireless networks"]}`),
			want: true,
		},
		{
			name: "unrelated meraki error",
			err:  errors.New(`rate limit exceeded`),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonWirelessNetworkError(tt.err); got != tt.want {
				t.Errorf("isNonWirelessNetworkError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
