package cmdutils

import "testing"

func TestParseSetSiteArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		want    ParsedSetSiteArgs
	}{
		{
			name: "config field",
			args: []string{"ap", "AP-01", "radio_config.band_5.channel", "36"},
			want: ParsedSetSiteArgs{Action: SetActionConfigField, Scope: ScopeSingle, DeviceType: "ap", Name: "AP-01", KeyPath: "radio_config.band_5.channel", RawValue: "36"},
		},
		{
			name: "arm single",
			args: []string{"ap", "AP-01", "managed"},
			want: ParsedSetSiteArgs{Action: SetActionArm, Scope: ScopeSingle, DeviceType: "ap", Name: "AP-01"},
		},
		{
			name: "disarm single",
			args: []string{"switch", "SW-01", "unmanaged"},
			want: ParsedSetSiteArgs{Action: SetActionDisarm, Scope: ScopeSingle, DeviceType: "switch", Name: "SW-01"},
		},
		{
			name: "arm all of type",
			args: []string{"ap", "all", "managed"},
			want: ParsedSetSiteArgs{Action: SetActionArm, Scope: ScopeAllOfType, DeviceType: "ap"},
		},
		{
			name: "disarm all types",
			args: []string{"all", "unmanaged"},
			want: ParsedSetSiteArgs{Action: SetActionDisarm, Scope: ScopeAllTypes},
		},
		{
			name: "device-type alias normalizes",
			args: []string{"gw", "all", "managed"},
			want: ParsedSetSiteArgs{Action: SetActionArm, Scope: ScopeAllOfType, DeviceType: "gateway"},
		},
		{
			// 4 tokens is always config-field, so a key-path literally named
			// "managed" with a value still works.
			name: "managed as key-path with value",
			args: []string{"ap", "AP-01", "managed", "true"},
			want: ParsedSetSiteArgs{Action: SetActionConfigField, Scope: ScopeSingle, DeviceType: "ap", Name: "AP-01", KeyPath: "managed", RawValue: "true"},
		},
		{name: "empty", args: nil, wantErr: true},
		{name: "bad device type", args: []string{"router", "R-01", "managed"}, wantErr: true},
		{name: "all without keyword", args: []string{"all"}, wantErr: true},
		{name: "all with bad keyword", args: []string{"all", "on"}, wantErr: true},
		{name: "bulk config rejected", args: []string{"ap", "all", "key", "val"}, wantErr: true},
		{name: "type only", args: []string{"ap"}, wantErr: true},
		{name: "single trailing junk", args: []string{"ap", "AP-01", "key", "val", "extra"}, wantErr: true},
		{name: "single bad keyword", args: []string{"ap", "AP-01", "online"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSetSiteArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestIsDeviceType(t *testing.T) {
	for _, s := range []string{"ap", "aps", "switch", "switches", "sw", "gateway", "gateways", "gw"} {
		if !IsDeviceType(s) {
			t.Errorf("IsDeviceType(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"all", "router", "", "AP"} {
		if IsDeviceType(s) {
			t.Errorf("IsDeviceType(%q) = true, want false", s)
		}
	}
}
