package cmdutils

import (
	"testing"
)

func TestParseSearchArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want SearchArgs
	}{
		{
			name: "empty",
			args: nil,
			want: SearchArgs{Format: "table"},
		},
		{
			name: "plain search text",
			args: []string{"laptop"},
			want: SearchArgs{SearchText: "laptop", Format: "table"},
		},
		{
			name: "text with site name",
			args: []string{"laptop", "site", "US-LAB-01"},
			want: SearchArgs{SearchText: "laptop", SiteID: "US-LAB-01", Format: "table"},
		},
		{
			name: "site first, no search text",
			args: []string{"site", "US-LAB-01"},
			want: SearchArgs{SiteID: "US-LAB-01", Format: "table"},
		},
		{
			name: "site first with multi-word quoted name",
			args: []string{"site", "MX - Av. Ejercito Nacional Mexicano 904"},
			want: SearchArgs{SiteID: "MX - Av. Ejercito Nacional Mexicano 904", Format: "table"},
		},
		{
			name: "site arg surrounded by stray quotes is stripped",
			args: []string{"site", `"quoted-site"`},
			want: SearchArgs{SiteID: "quoted-site", Format: "table"},
		},
		{
			name: "site first with json format",
			args: []string{"site", "L_3732358191183298569", "json"},
			want: SearchArgs{SiteID: "L_3732358191183298569", Format: "json"},
		},
		{
			name: "empty string then site",
			args: []string{"", "site", "US-LAB-01"},
			want: SearchArgs{SiteID: "US-LAB-01", Format: "table"},
		},
		{
			name: "force flag",
			args: []string{"laptop", "force"},
			want: SearchArgs{SearchText: "laptop", Force: true, Format: "table"},
		},
		{
			name: "csv format",
			args: []string{"laptop", "csv"},
			want: SearchArgs{SearchText: "laptop", Format: "csv"},
		},
		{
			name: "no-resolve",
			args: []string{"laptop", "no-resolve"},
			want: SearchArgs{SearchText: "laptop", Format: "table", NoResolve: true},
		},
		{
			name: "MAC address and site",
			args: []string{"aa:bb:cc:dd:ee:ff", "site", "US-LAB-01", "json"},
			want: SearchArgs{SearchText: "aa:bb:cc:dd:ee:ff", SiteID: "US-LAB-01", Format: "json"},
		},
		{
			name: "detail keyword alone",
			args: []string{"laptop", "detail"},
			want: SearchArgs{SearchText: "laptop", Format: "table", Detail: true},
		},
		{
			name: "site-scoped detail",
			args: []string{"site", "US-LAB-01", "detail"},
			want: SearchArgs{SiteID: "US-LAB-01", Format: "table", Detail: true},
		},
		{
			name: "extensive keyword",
			args: []string{"site", "US-LAB-01", "extensive"},
			want: SearchArgs{SiteID: "US-LAB-01", Format: "table", Extensive: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSearchArgs(tt.args)
			if got != tt.want {
				t.Errorf("ParseSearchArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestValidateSearchArgs(t *testing.T) {
	tests := []struct {
		name    string
		parsed  SearchArgs
		wantErr bool
	}{
		{name: "search text only", parsed: SearchArgs{SearchText: "laptop"}, wantErr: false},
		{name: "site only", parsed: SearchArgs{SiteID: "US-LAB-01"}, wantErr: false},
		{name: "both", parsed: SearchArgs{SearchText: "laptop", SiteID: "US-LAB-01"}, wantErr: false},
		{name: "neither", parsed: SearchArgs{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSearchArgs(tt.parsed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSearchArgs(%+v) err = %v, wantErr = %v", tt.parsed, err, tt.wantErr)
			}
		})
	}
}

func TestStripQuotes(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`"quoted"`, "quoted"},
		{`unquoted`, "unquoted"},
		{`"`, `"`},
		{``, ``},
		{`"missing-end`, `"missing-end`},
		{`missing-start"`, `missing-start"`},
	}
	for _, tt := range tests {
		if got := StripQuotes(tt.in); got != tt.want {
			t.Errorf("StripQuotes(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
