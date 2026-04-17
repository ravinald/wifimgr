package cmd

import "testing"

func TestParseRefreshClientSiteArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantSite string
		wantAPI  string
		wantErr  bool
	}{
		{
			name:     "bare site name",
			args:     []string{"US-LAB-01"},
			wantSite: "US-LAB-01",
		},
		{
			name:     "site keyword + name",
			args:     []string{"site", "US-LAB-01"},
			wantSite: "US-LAB-01",
		},
		{
			name:     "name + target",
			args:     []string{"US-LAB-01", "target", "meraki-corp"},
			wantSite: "US-LAB-01",
			wantAPI:  "meraki-corp",
		},
		{
			name:     "site keyword + quoted multi-word name",
			args:     []string{"site", "MX - Av. Ejercito Nacional Mexicano 904"},
			wantSite: "MX - Av. Ejercito Nacional Mexicano 904",
		},
		{
			name:    "missing site name",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "target without value",
			args:    []string{"US-LAB-01", "target"},
			wantErr: true,
		},
		{
			name:    "unexpected trailing arg",
			args:    []string{"US-LAB-01", "refresh-me-please"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRefreshClientSiteArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got.siteName != tt.wantSite {
				t.Errorf("siteName = %q, want %q", got.siteName, tt.wantSite)
			}
			if got.target != tt.wantAPI {
				t.Errorf("target = %q, want %q", got.target, tt.wantAPI)
			}
		})
	}
}
