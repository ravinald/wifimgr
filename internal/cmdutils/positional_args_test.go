package cmdutils

import (
	"strings"
	"testing"
)

func TestParseShowArgsFormat(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantFormat string
		wantErr    string // substring; empty means no error
	}{
		{name: "default is table", args: nil, wantFormat: "table"},
		{name: "format json", args: []string{"format", "json"}, wantFormat: "json"},
		{name: "format csv", args: []string{"format", "csv"}, wantFormat: "csv"},
		{name: "format alias", args: []string{"format", "alias"}, wantFormat: "alias"},
		{name: "format uppercased value", args: []string{"format", "JSON"}, wantFormat: "json"},
		{name: "bare json rejected", args: []string{"json"}, wantErr: "use 'format json'"},
		{name: "bare csv rejected", args: []string{"csv"}, wantErr: "use 'format csv'"},
		{name: "bare alias rejected", args: []string{"alias"}, wantErr: "use 'format alias'"},
		{name: "bare table rejected", args: []string{"table"}, wantErr: "use 'format table'"},
		{name: "invalid format value", args: []string{"format", "bogus"}, wantErr: "must be 'json', 'csv', or 'alias'"},
		{name: "format without value", args: []string{"format"}, wantErr: "requires a format type"},
		{name: "format specified twice", args: []string{"format", "json", "format", "csv"}, wantErr: "specified multiple times"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseShowArgs(tt.args)
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
			if parsed.Format != tt.wantFormat {
				t.Fatalf("Format = %q, want %q", parsed.Format, tt.wantFormat)
			}
		})
	}
}

func TestValidateShowArgsAliasScope(t *testing.T) {
	aliasArgs := []string{"format", "alias"}

	if err := ValidateShowAPArgs(nil, aliasArgs); err == nil {
		t.Error("ValidateShowAPArgs should reject alias format")
	}
	if err := ValidateInventoryArgs(nil, aliasArgs); err == nil {
		t.Error("ValidateInventoryArgs should reject alias format")
	}
	if err := ValidateShowBSSIDArgs(nil, aliasArgs); err != nil {
		t.Errorf("ValidateShowBSSIDArgs should accept alias format, got: %v", err)
	}

	// json stays valid everywhere.
	jsonArgs := []string{"format", "json"}
	if err := ValidateShowAPArgs(nil, jsonArgs); err != nil {
		t.Errorf("ValidateShowAPArgs rejected json: %v", err)
	}
	if err := ValidateShowBSSIDArgs(nil, jsonArgs); err != nil {
		t.Errorf("ValidateShowBSSIDArgs rejected json: %v", err)
	}
}
