/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"strings"
	"testing"
)

func TestValidatePSK(t *testing.T) {
	tests := []struct {
		name    string
		psk     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 8 char PSK",
			psk:     "12345678",
			wantErr: false,
		},
		{
			name:    "valid 63 char PSK",
			psk:     strings.Repeat("a", 63),
			wantErr: false,
		},
		{
			name:    "valid PSK with special chars",
			psk:     "MyP@ssw0rd!#$%",
			wantErr: false,
		},
		{
			name:    "valid PSK with spaces",
			psk:     "my wifi password",
			wantErr: false,
		},
		{
			name:    "too short - 7 chars",
			psk:     "1234567",
			wantErr: true,
			errMsg:  "at least 8 characters",
		},
		{
			name:    "too long - 64 chars",
			psk:     strings.Repeat("a", 64),
			wantErr: true,
			errMsg:  "at most 63 characters",
		},
		{
			name:    "empty PSK",
			psk:     "",
			wantErr: true,
			errMsg:  "at least 8 characters",
		},
		{
			name:    "contains non-printable char (tab)",
			psk:     "pass\tword",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "contains non-printable char (newline)",
			psk:     "pass\nword",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "contains non-ASCII char",
			psk:     "pässwörd",
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePSK(tt.psk)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePSK() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePSK() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePSK() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		isPSK   bool
		wantErr bool
	}{
		{
			name:    "generic secret - any non-empty value",
			secret:  "x",
			isPSK:   false,
			wantErr: false,
		},
		{
			name:    "generic secret - empty fails",
			secret:  "",
			isPSK:   false,
			wantErr: true,
		},
		{
			name:    "PSK mode - valid",
			secret:  "validpsk",
			isPSK:   true,
			wantErr: false,
		},
		{
			name:    "PSK mode - too short",
			secret:  "short",
			isPSK:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecret(tt.secret, tt.isPSK)
			if tt.wantErr && err == nil {
				t.Errorf("validateSecret() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateSecret() unexpected error: %v", err)
			}
		})
	}
}
