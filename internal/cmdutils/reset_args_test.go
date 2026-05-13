/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmdutils

import (
	"strings"
	"testing"
)

func TestParseResetArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantAP    string
		wantSite  string
		wantForce bool
		wantErr   string // substring; "" means no error
	}{
		{
			name:   "ap name only",
			args:   []string{"AP-LAB-01"},
			wantAP: "AP-LAB-01",
		},
		{
			name:     "ap name with site",
			args:     []string{"AP-LAB-01", "site", "US-LAB-01"},
			wantAP:   "AP-LAB-01",
			wantSite: "US-LAB-01",
		},
		{
			name:      "ap name with force",
			args:      []string{"AP-LAB-01", "force"},
			wantAP:    "AP-LAB-01",
			wantForce: true,
		},
		{
			name:      "ap site force",
			args:      []string{"AP-LAB-01", "site", "US-LAB-01", "force"},
			wantAP:    "AP-LAB-01",
			wantSite:  "US-LAB-01",
			wantForce: true,
		},
		{
			name:      "ap force site (any keyword order)",
			args:      []string{"AP-LAB-01", "force", "site", "US-LAB-01"},
			wantAP:    "AP-LAB-01",
			wantSite:  "US-LAB-01",
			wantForce: true,
		},
		{
			name:     "quoted ap and site",
			args:     []string{`"AP LAB 01"`, "site", `"US LAB 01"`},
			wantAP:   "AP LAB 01",
			wantSite: "US LAB 01",
		},
		{
			name:      "case-insensitive site keyword",
			args:      []string{"AP-LAB-01", "SITE", "US-LAB-01", "FORCE"},
			wantAP:    "AP-LAB-01",
			wantSite:  "US-LAB-01",
			wantForce: true,
		},

		// Error cases
		{
			name:    "no args",
			args:    nil,
			wantErr: "missing AP name",
		},
		{
			name:    "site missing value",
			args:    []string{"AP-LAB-01", "site"},
			wantErr: "'site' requires a site name",
		},
		{
			name:    "duplicate site",
			args:    []string{"AP-LAB-01", "site", "A", "site", "B"},
			wantErr: "site specified multiple times",
		},
		{
			name:    "duplicate force",
			args:    []string{"AP-LAB-01", "force", "force"},
			wantErr: "'force' specified multiple times",
		},
		{
			name:    "unexpected trailing token",
			args:    []string{"AP-LAB-01", "json"},
			wantErr: "unexpected positional",
		},
		{
			name:    "flag-style arg rejected",
			args:    []string{"AP-LAB-01", "--force"},
			wantErr: "unexpected positional",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResetArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.APName != tt.wantAP {
				t.Errorf("APName: got %q, want %q", got.APName, tt.wantAP)
			}
			if got.SiteName != tt.wantSite {
				t.Errorf("SiteName: got %q, want %q", got.SiteName, tt.wantSite)
			}
			if got.Force != tt.wantForce {
				t.Errorf("Force: got %v, want %v", got.Force, tt.wantForce)
			}
		})
	}
}
