package meraki

import (
	"testing"
)

// TestExtractMerakiRadioBody_PassesThroughFullShape proves the radio push is not
// truncated to the SDK's typed 2.4/5 GHz subset — 6 GHz, flex-radio, per-SSID, and
// bitrate fields survive because the raw radio_settings block passes through verbatim.
func TestExtractMerakiRadioBody_PassesThroughFullShape(t *testing.T) {
	config := map[string]any{
		"name": "ap-1",
		"radio_settings": map[string]any{
			"serial":             "Q2ZD-BQ32-KPNP", // read-only echo, must be dropped
			"rfProfileId":        "12345",
			"twoFourGhzSettings": map[string]any{"channel": 6, "targetPower": 12},
			"fiveGhzSettings":    map[string]any{"channel": 149, "channelWidth": 80, "targetPower": 15},
			"sixGhzSettings":     map[string]any{"channel": 37, "channelWidth": 160, "targetPower": 18},
			"flexRadioBand":      "six",
			"perSsidSettings":    map[string]any{"0": map[string]any{"minBitrate": 12}},
		},
	}

	body := extractMerakiRadioBody(config)

	for _, key := range []string{"sixGhzSettings", "flexRadioBand", "perSsidSettings", "rfProfileId", "twoFourGhzSettings", "fiveGhzSettings"} {
		if _, ok := body[key]; !ok {
			t.Errorf("radio body dropped %q — full shape not preserved", key)
		}
	}
	if _, ok := body["serial"]; ok {
		t.Error("radio body should drop the read-only serial echo")
	}
}

func TestExtractMerakiRadioBody_DropsEmptyBands(t *testing.T) {
	config := map[string]any{
		"radio_settings": map[string]any{
			"serial":             "Q2ZD-BQ32-KPNP",
			"fiveGhzSettings":    map[string]any{"channel": 149},
			"twoFourGhzSettings": map[string]any{}, // unset band, do not send
		},
	}

	body := extractMerakiRadioBody(config)
	if _, ok := body["twoFourGhzSettings"]; ok {
		t.Error("empty 2.4 GHz block should be dropped to avoid resetting the band")
	}
	if _, ok := body["fiveGhzSettings"]; !ok {
		t.Error("set 5 GHz block should be kept")
	}
}

func TestExtractMerakiRadioBody_TranslatesAgnostic(t *testing.T) {
	config := map[string]any{
		"radio_config": map[string]any{
			"band_5":  map[string]any{"channel": 36, "power": 14},
			"band_24": map[string]any{"channel": 6, "power": 11},
		},
	}

	body := extractMerakiRadioBody(config)
	five, ok := body["fiveGhzSettings"].(map[string]any)
	if !ok {
		t.Fatalf("agnostic radio_config should translate to fiveGhzSettings, got %v", body)
	}
	if five["channel"] != 36 {
		t.Errorf("fiveGhzSettings.channel = %v, want 36", five["channel"])
	}
	if five["targetPower"] != 14 {
		t.Errorf("fiveGhzSettings.targetPower = %v, want 14", five["targetPower"])
	}
}

func TestExtractMerakiRadioBody_NoneWhenAbsentOrEmpty(t *testing.T) {
	if body := extractMerakiRadioBody(map[string]any{"name": "ap-1"}); body != nil {
		t.Errorf("no radio in config should yield nil body, got %v", body)
	}
	empty := map[string]any{"radio_settings": map[string]any{"serial": "Q2ZD-BQ32-KPNP"}}
	if body := extractMerakiRadioBody(empty); body != nil {
		t.Errorf("radio_settings with only a serial echo should yield nil body, got %v", body)
	}
}

// TestBuildRadioRequest_TypedSubsetAndDeferral proves the SDK's typed request carries
// 2.4/5 GHz + rfProfile, and that 6 GHz/flex/per-SSID surface as deferred (logged, not
// silently dropped) until the bolt-on lands.
func TestBuildRadioRequest_TypedSubsetAndDeferral(t *testing.T) {
	body := map[string]any{
		"rfProfileId":        "12345",
		"twoFourGhzSettings": map[string]any{"channel": 6, "targetPower": 12},
		"fiveGhzSettings":    map[string]any{"channel": 149, "channelWidth": "80", "targetPower": 15},
		"sixGhzSettings":     map[string]any{"channel": 37},
		"flexRadioBand":      "six",
	}

	req, deferred := buildRadioRequest(body)
	if req == nil {
		t.Fatal("request should be populated from 2.4/5 GHz + rfProfile")
	}
	if req.RfProfileID != "12345" {
		t.Errorf("RfProfileID = %q, want 12345", req.RfProfileID)
	}
	if req.TwoFourGhzSettings == nil || req.TwoFourGhzSettings.Channel == nil || *req.TwoFourGhzSettings.Channel != 6 {
		t.Errorf("2.4 GHz channel not mapped: %+v", req.TwoFourGhzSettings)
	}
	// channelWidth arrives as the translator's string "80" — must coerce to int 80.
	if req.FiveGhzSettings == nil || req.FiveGhzSettings.ChannelWidth == nil || *req.FiveGhzSettings.ChannelWidth != 80 {
		t.Errorf("5 GHz channelWidth not coerced to 80: %+v", req.FiveGhzSettings)
	}

	want := map[string]bool{"sixGhzSettings": true, "flexRadioBand": true}
	if len(deferred) != len(want) {
		t.Fatalf("deferred = %v, want keys %v", deferred, want)
	}
	for _, k := range deferred {
		if !want[k] {
			t.Errorf("unexpected deferred field %q", k)
		}
	}
}

func TestBuildRadioRequest_NilWhenOnlyDeferred(t *testing.T) {
	req, deferred := buildRadioRequest(map[string]any{"sixGhzSettings": map[string]any{"channel": 37}})
	if req != nil {
		t.Error("request should be nil when only 6 GHz is present (nothing the typed request can send)")
	}
	if len(deferred) != 1 || deferred[0] != "sixGhzSettings" {
		t.Errorf("deferred = %v, want [sixGhzSettings]", deferred)
	}
}

func TestToIntPtr(t *testing.T) {
	cases := []struct {
		in   any
		want *int
	}{
		{float64(80), ptr(80)}, // JSON decode
		{int(36), ptr(36)},     // translator
		{"160", ptr(160)},      // translator channelWidth
		{"auto", nil},          // unparseable
		{nil, nil},
	}
	for _, c := range cases {
		got := toIntPtr(c.in)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("toIntPtr(%v) = %v, want %v", c.in, derefOrNil(got), derefOrNil(c.want))
		}
	}
}

func ptr(i int) *int { return &i }

func derefOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestBuildDeviceFieldUpdate_RadioOnlySkipsDevicePut(t *testing.T) {
	_, has := buildDeviceFieldUpdate(map[string]any{
		"radio_settings": map[string]any{"fiveGhzSettings": map[string]any{"channel": 149}},
	})
	if has {
		t.Error("a radio-only config should not trigger a device-attributes PUT")
	}
	if _, has := buildDeviceFieldUpdate(map[string]any{"name": "ap-1"}); !has {
		t.Error("a name change should trigger a device-attributes PUT")
	}
}
