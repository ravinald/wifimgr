package cmd

import (
	"testing"
)

func TestParseSearchArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want searchArgs
	}{
		{
			name: "empty",
			args: nil,
			want: searchArgs{format: "table"},
		},
		{
			name: "plain search text",
			args: []string{"laptop"},
			want: searchArgs{searchText: "laptop", format: "table"},
		},
		{
			name: "text with site name",
			args: []string{"laptop", "site", "US-LAB-01"},
			want: searchArgs{searchText: "laptop", siteID: "US-LAB-01", format: "table"},
		},
		{
			name: "site first, no search text",
			args: []string{"site", "US-LAB-01"},
			want: searchArgs{siteID: "US-LAB-01", format: "table"},
		},
		{
			name: "site first with multi-word quoted name",
			args: []string{"site", "MX - Av. Ejercito Nacional Mexicano 904"},
			want: searchArgs{siteID: "MX - Av. Ejercito Nacional Mexicano 904", format: "table"},
		},
		{
			name: "site arg surrounded by stray quotes is stripped",
			args: []string{"site", `"quoted-site"`},
			want: searchArgs{siteID: "quoted-site", format: "table"},
		},
		{
			name: "site first with json format",
			args: []string{"site", "L_3732358191183298569", "json"},
			want: searchArgs{siteID: "L_3732358191183298569", format: "json"},
		},
		{
			name: "empty string then site",
			args: []string{"", "site", "US-LAB-01"},
			want: searchArgs{siteID: "US-LAB-01", format: "table"},
		},
		{
			name: "force flag",
			args: []string{"laptop", "force"},
			want: searchArgs{searchText: "laptop", force: true, format: "table"},
		},
		{
			name: "csv format",
			args: []string{"laptop", "csv"},
			want: searchArgs{searchText: "laptop", format: "csv"},
		},
		{
			name: "no-resolve",
			args: []string{"laptop", "no-resolve"},
			want: searchArgs{searchText: "laptop", format: "table", noResolve: true},
		},
		{
			name: "MAC address and site",
			args: []string{"aa:bb:cc:dd:ee:ff", "site", "US-LAB-01", "json"},
			want: searchArgs{searchText: "aa:bb:cc:dd:ee:ff", siteID: "US-LAB-01", format: "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSearchArgs(tt.args)
			if got != tt.want {
				t.Errorf("parseSearchArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestValidateSearchArgs(t *testing.T) {
	tests := []struct {
		name    string
		parsed  searchArgs
		wantErr bool
	}{
		{name: "search text only", parsed: searchArgs{searchText: "laptop"}, wantErr: false},
		{name: "site only", parsed: searchArgs{siteID: "US-LAB-01"}, wantErr: false},
		{name: "both", parsed: searchArgs{searchText: "laptop", siteID: "US-LAB-01"}, wantErr: false},
		{name: "neither", parsed: searchArgs{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSearchArgs(tt.parsed)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSearchArgs(%+v) err = %v, wantErr = %v", tt.parsed, err, tt.wantErr)
			}
		})
	}
}

func TestSearchStripQuotes(t *testing.T) {
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
		if got := searchStripQuotes(tt.in); got != tt.want {
			t.Errorf("searchStripQuotes(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
