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

func TestParseRefreshArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		opts     ParseRefreshOptions
		wantAPI  string
		wantSite string
		wantErr  string // substring; "" means no error expected
	}{
		// Empty input
		{
			name: "empty args, site allowed",
			args: nil,
			opts: ParseRefreshOptions{AllowSite: true},
		},
		{
			name: "empty args, site disallowed",
			args: nil,
			opts: ParseRefreshOptions{},
		},

		// api keyword
		{
			name:    "api <name>",
			args:    []string{"api", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantAPI: "meraki-corp",
		},
		{
			name:    "api <name> with quoted value",
			args:    []string{"api", `"mist-prod"`},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantAPI: "mist-prod",
		},
		{
			name:    "api missing value",
			args:    []string{"api"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'api' requires an API label",
		},
		{
			name:    "api specified twice",
			args:    []string{"api", "a", "api", "b"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "api specified multiple times",
		},

		// site keyword
		{
			name:     "site <name>",
			args:     []string{"site", "US-LAB-01"},
			opts:     ParseRefreshOptions{AllowSite: true},
			wantSite: "US-LAB-01",
		},
		{
			name:     "site <name> api <api>",
			args:     []string{"site", "US-LAB-01", "api", "meraki-corp"},
			opts:     ParseRefreshOptions{AllowSite: true},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:     "api <api> site <name> (order reversed)",
			args:     []string{"api", "meraki-corp", "site", "US-LAB-01"},
			opts:     ParseRefreshOptions{AllowSite: true},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:    "site missing value",
			args:    []string{"site"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'site' requires a site name",
		},
		{
			name:    "site rejected when not allowed",
			args:    []string{"site", "US-LAB-01"},
			opts:    ParseRefreshOptions{},
			wantErr: "'site' is not valid here",
		},
		{
			name:    "site specified twice",
			args:    []string{"site", "a", "site", "b"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "site specified multiple times",
		},

		// target migration
		{
			name:    "target keyword rejected with migration hint",
			args:    []string{"site", "US-LAB-01", "target", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'target' keyword has been removed",
		},
		{
			name:    "target as leading keyword also rejected",
			args:    []string{"target", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'target' keyword has been removed",
		},

		// hard break: bare positional
		{
			name:    "bare api name without keyword (refresh device hard break)",
			args:    []string{"meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "did you mean 'api meraki-corp'",
		},
		{
			name:    "trailing unknown token after valid form",
			args:    []string{"api", "meraki-corp", "junk"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "did you mean 'api junk'",
		},

		// implicit site (refresh client site)
		{
			name:     "implicit site as leading token",
			args:     []string{"US-LAB-01"},
			opts:     ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantSite: "US-LAB-01",
		},
		{
			name:     "implicit site + api keyword",
			args:     []string{"US-LAB-01", "api", "meraki-corp"},
			opts:     ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:     "implicit site + explicit site keyword still works",
			args:     []string{"site", "US-LAB-01", "api", "meraki-corp"},
			opts:     ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:    "implicit site + target rejected",
			args:    []string{"US-LAB-01", "target", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantErr: "'target' keyword has been removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRefreshArgs(tt.args, tt.opts)

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
			if got.APIName != tt.wantAPI {
				t.Errorf("APIName = %q, want %q", got.APIName, tt.wantAPI)
			}
			if got.SiteName != tt.wantSite {
				t.Errorf("SiteName = %q, want %q", got.SiteName, tt.wantSite)
			}
		})
	}
}
