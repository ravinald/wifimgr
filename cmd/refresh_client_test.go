/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"strings"
	"testing"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// These tests cover the args-parsing surface used by `refresh client site`.
// The parser itself lives in internal/cmdutils and has its own table-driven
// coverage; this file pins the specific options combo and migration error
// behaviour that this command relies on.
func TestRefreshClientSiteArgParsing(t *testing.T) {
	opts := cmdutils.ParseRefreshOptions{
		AllowSite:         true,
		AllowImplicitSite: true,
	}

	tests := []struct {
		name     string
		args     []string
		wantSite string
		wantAPI  string
		wantErr  string // substring; empty means no error
	}{
		{
			name:     "bare site name",
			args:     []string{"US-LAB-01"},
			wantSite: "US-LAB-01",
		},
		{
			name:     "explicit site keyword + name",
			args:     []string{"site", "US-LAB-01"},
			wantSite: "US-LAB-01",
		},
		{
			name:     "site name + api keyword",
			args:     []string{"US-LAB-01", "api", "meraki-corp"},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:     "site keyword + multi-word name",
			args:     []string{"site", "MX - Av. Ejercito Nacional Mexicano 904"},
			wantSite: "MX - Av. Ejercito Nacional Mexicano 904",
		},
		{
			name:    "legacy target keyword now a hard break",
			args:    []string{"US-LAB-01", "target", "meraki-corp"},
			wantErr: "'target' keyword has been removed",
		},
		{
			name:    "api without value",
			args:    []string{"US-LAB-01", "api"},
			wantErr: "'api' requires an API label",
		},
		{
			name:    "unexpected trailing token",
			args:    []string{"US-LAB-01", "refresh-me-please"},
			wantErr: "did you mean 'api refresh-me-please'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cmdutils.ParseRefreshArgs(tt.args, opts)

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
			if got.SiteName != tt.wantSite {
				t.Errorf("SiteName = %q, want %q", got.SiteName, tt.wantSite)
			}
			if got.APIName != tt.wantAPI {
				t.Errorf("APIName = %q, want %q", got.APIName, tt.wantAPI)
			}
		})
	}
}
