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
		name       string
		args       []string
		opts       ParseRefreshOptions
		wantTarget string
		wantSite   string
		wantScope  string
		wantErr    string // substring; "" means no error expected
	}{
		// Empty input
		{
			name: "empty args, site allowed",
			args: nil,
			opts: ParseRefreshOptions{AllowSite: true, AllowScope: true},
		},
		{
			name: "empty args, site disallowed",
			args: nil,
			opts: ParseRefreshOptions{},
		},

		// target keyword (the API selector)
		{
			name:       "target <name>",
			args:       []string{"target", "meraki-corp"},
			opts:       ParseRefreshOptions{AllowSite: true, AllowScope: true},
			wantTarget: "meraki-corp",
		},
		{
			name:       "target <name> with quoted value",
			args:       []string{"target", `"mist-prod"`},
			opts:       ParseRefreshOptions{AllowSite: true},
			wantTarget: "mist-prod",
		},
		{
			name:    "target missing value",
			args:    []string{"target"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'target' requires an API label",
		},
		{
			name:    "target specified twice",
			args:    []string{"target", "a", "target", "b"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "target specified multiple times",
		},

		// scope words
		{
			name:      "all scope",
			args:      []string{"all"},
			opts:      ParseRefreshOptions{AllowSite: true, AllowScope: true},
			wantScope: "all",
		},
		{
			name:      "detail scope",
			args:      []string{"detail"},
			opts:      ParseRefreshOptions{AllowSite: true, AllowScope: true},
			wantScope: "detail",
		},
		{
			name:      "site with all scope",
			args:      []string{"site", "US-LAB-01", "all"},
			opts:      ParseRefreshOptions{AllowSite: true, AllowScope: true},
			wantSite:  "US-LAB-01",
			wantScope: "all",
		},
		{
			name:    "scope rejected when not allowed",
			args:    []string{"all"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: `"all" is not valid here`,
		},
		{
			name:    "scope specified twice",
			args:    []string{"all", "detail"},
			opts:    ParseRefreshOptions{AllowSite: true, AllowScope: true},
			wantErr: "scope specified multiple times",
		},

		// site keyword
		{
			name:     "site <name>",
			args:     []string{"site", "US-LAB-01"},
			opts:     ParseRefreshOptions{AllowSite: true},
			wantSite: "US-LAB-01",
		},
		{
			name:       "site <name> target <api>",
			args:       []string{"site", "US-LAB-01", "target", "meraki-corp"},
			opts:       ParseRefreshOptions{AllowSite: true},
			wantSite:   "US-LAB-01",
			wantTarget: "meraki-corp",
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

		// api migration
		{
			name:    "api keyword rejected with migration hint",
			args:    []string{"site", "US-LAB-01", "api", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'api' keyword has been removed",
		},
		{
			name:    "api as leading keyword also rejected",
			args:    []string{"api", "meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: "'api' keyword has been removed",
		},

		// bare positional
		{
			name:    "bare token without keyword",
			args:    []string{"meraki-corp"},
			opts:    ParseRefreshOptions{AllowSite: true},
			wantErr: `unexpected positional "meraki-corp"`,
		},

		// implicit site (refresh client site)
		{
			name:     "implicit site as leading token",
			args:     []string{"US-LAB-01"},
			opts:     ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantSite: "US-LAB-01",
		},
		{
			name:       "implicit site + target keyword",
			args:       []string{"US-LAB-01", "target", "meraki-corp"},
			opts:       ParseRefreshOptions{AllowSite: true, AllowImplicitSite: true},
			wantSite:   "US-LAB-01",
			wantTarget: "meraki-corp",
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
			if got.Target != tt.wantTarget {
				t.Errorf("Target = %q, want %q", got.Target, tt.wantTarget)
			}
			if got.SiteName != tt.wantSite {
				t.Errorf("SiteName = %q, want %q", got.SiteName, tt.wantSite)
			}
			if got.Scope != tt.wantScope {
				t.Errorf("Scope = %q, want %q", got.Scope, tt.wantScope)
			}
		})
	}
}
