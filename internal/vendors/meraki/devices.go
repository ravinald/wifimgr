package meraki

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// devicesService implements vendors.DevicesService for Meraki.
type devicesService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List returns devices in a network, optionally filtered by type.
func (s *devicesService) List(ctx context.Context, siteID, deviceType string) ([]*vendors.DeviceInfo, error) {
	logging.Debugf("[meraki] Listing devices for org %s, siteID=%q, deviceType=%q", s.orgID, siteID, deviceType)

	params := &meraki.GetOrganizationDevicesQueryParams{
		PerPage: -1, // Fetch all
	}

	if siteID != "" {
		params.NetworkIDs = []string{siteID}
	}

	if deviceType != "" {
		productType := mapDeviceTypeToProductType(deviceType)
		params.ProductTypes = []string{productType}
	}

	retryState := NewRetryState(s.retryConfig)
	var devices *meraki.ResponseOrganizationsGetOrganizationDevices
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var resp *meraki.ResponseOrganizationsGetOrganizationDevices
		var httpResp *resty.Response
		if s.suppressOutput {
			restore := suppressStdout()
			resp, httpResp, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
			restore()
		} else {
			resp, httpResp, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
		}
		devices = resp

		// Classify transport failures once at the boundary. Everything
		// upstream reads the wifimgr taxonomy via errors.As.
		err = ClassifyError(s.orgID, "GetOrganizationDevices", httpResp, err)

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get devices: %v", err)
			return nil, err
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if devices == nil {
		logging.Debug("[meraki] No devices returned")
		return []*vendors.DeviceInfo{}, nil
	}

	infos := make([]*vendors.DeviceInfo, 0, len(*devices))
	for i := range *devices {
		info := convertDeviceToDeviceInfo(&(*devices)[i])
		if info != nil {
			infos = append(infos, info)
		}
	}

	logging.Debugf("[meraki] Listed %d devices", len(infos))
	return infos, nil
}

// ByMAC finds a device by MAC address.
func (s *devicesService) ByMAC(ctx context.Context, mac string) (*vendors.DeviceInfo, error) {
	params := &meraki.GetOrganizationDevicesQueryParams{
		PerPage: -1,
	}

	retryState := NewRetryState(s.retryConfig)
	var devices *meraki.ResponseOrganizationsGetOrganizationDevices
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var httpResp *resty.Response
		devices, httpResp, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
		err = ClassifyError(s.orgID, "GetOrganizationDevices", httpResp, err)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, err
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if devices == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: mac}
	}

	normalizedMAC := normalizeMAC(mac)
	for i := range *devices {
		if normalizeMAC((*devices)[i].Mac) == normalizedMAC {
			return convertDeviceToDeviceInfo(&(*devices)[i]), nil
		}
	}

	return nil, &vendors.DeviceNotFoundError{Identifier: mac}
}

// Get finds a device by its serial (Meraki uses serial as device ID).
func (s *devicesService) Get(ctx context.Context, _, deviceID string) (*vendors.DeviceInfo, error) {
	// In Meraki, deviceID is the serial number
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var httpResp *resty.Response
		device, httpResp, err = s.dashboard.Devices.GetDevice(deviceID)
		err = ClassifyError(s.orgID, "GetDevice", httpResp, err)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, err
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if device == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: deviceID}
	}

	return &vendors.DeviceInfo{
		ID:           device.Serial,
		MAC:          normalizeMAC(device.Mac),
		Serial:       device.Serial,
		Model:        device.Model,
		Name:         device.Name,
		SiteID:       device.NetworkID,
		Notes:        device.Notes,
		IP:           device.LanIP,
		Version:      device.Firmware,
		SourceVendor: "meraki",
	}, nil
}

// Update modifies a device's configuration.
func (s *devicesService) Update(ctx context.Context, _, deviceID string, device *vendors.DeviceInfo) (*vendors.DeviceInfo, error) {
	if device == nil {
		return nil, fmt.Errorf("device cannot be nil")
	}

	// deviceID in Meraki is the serial number
	request := &meraki.RequestDevicesUpdateDevice{
		Name:  device.Name,
		Notes: device.Notes,
	}

	// Handle optional fields
	if device.Latitude != 0 || device.Longitude != 0 {
		request.Lat = &device.Latitude
		request.Lng = &device.Longitude
	}

	retryState := NewRetryState(s.retryConfig)
	var updatedDevice *meraki.ResponseDevicesUpdateDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var httpResp *resty.Response
		updatedDevice, httpResp, err = s.dashboard.Devices.UpdateDevice(deviceID, request)
		err = ClassifyError(s.orgID, "UpdateDevice", httpResp, err)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, err
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return &vendors.DeviceInfo{
		ID:           updatedDevice.Serial,
		MAC:          normalizeMAC(updatedDevice.Mac),
		Serial:       updatedDevice.Serial,
		Model:        updatedDevice.Model,
		Name:         updatedDevice.Name,
		SiteID:       updatedDevice.NetworkID,
		Notes:        updatedDevice.Notes,
		IP:           updatedDevice.LanIP,
		Version:      updatedDevice.Firmware,
		SourceVendor: "meraki",
	}, nil
}

// Rename changes the device name.
func (s *devicesService) Rename(ctx context.Context, _, deviceID, newName string) error {
	// deviceID in Meraki is the serial number
	request := &meraki.RequestDevicesUpdateDevice{
		Name: newName,
	}

	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, _, err := s.dashboard.Devices.UpdateDevice(deviceID, request)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to rename device %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// UpdateConfig applies a raw configuration map to a device. Device-level fields
// (name/notes/geo) and radio settings live behind separate Meraki endpoints, so a
// config carrying both drives two calls; either alone drives one. Radio goes through
// the SDK's typed radio endpoint (2.4/5 GHz + rfProfile); 6 GHz/flex/per-SSID are
// deferred with a logged warning until the bolt-on lands (see applyRadioSettings).
func (s *devicesService) UpdateConfig(ctx context.Context, _, deviceID string, config map[string]interface{}) error {
	// deviceID in Meraki is the serial number.
	if request, has := buildDeviceFieldUpdate(config); has {
		if err := s.putDeviceUpdate(ctx, deviceID, request); err != nil {
			return err
		}
	}

	if body := extractMerakiRadioBody(config); len(body) > 0 {
		if err := s.applyRadioSettings(ctx, deviceID, body); err != nil {
			return err
		}
	}

	return nil
}

// buildDeviceFieldUpdate maps the device-level fields from a config map. The bool
// reports whether any field was set, so an update touching only radio skips the
// device-attributes PUT entirely.
func buildDeviceFieldUpdate(config map[string]interface{}) (*meraki.RequestDevicesUpdateDevice, bool) {
	request := &meraki.RequestDevicesUpdateDevice{}
	has := false

	if name, ok := config["name"].(string); ok {
		request.Name = name
		has = true
	}
	if notes, ok := config["notes"].(string); ok {
		request.Notes = notes
		has = true
	}
	if lat, ok := config["lat"].(float64); ok {
		request.Lat = &lat
		has = true
	}
	if lng, ok := config["lng"].(float64); ok {
		request.Lng = &lng
		has = true
	}
	if address, ok := config["address"].(string); ok {
		request.Address = address
		has = true
	}
	if floorPlanID, ok := config["floor_plan_id"].(string); ok {
		request.FloorPlanID = floorPlanID
		has = true
	}

	return request, has
}

// putDeviceUpdate sends device-level attributes through the SDK with the standard
// rate-limit + retry + error-classification pattern.
func (s *devicesService) putDeviceUpdate(ctx context.Context, deviceID string, request *meraki.RequestDevicesUpdateDevice) error {
	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, httpResp, err := s.dashboard.Devices.UpdateDevice(deviceID, request)
		err = ClassifyError(s.orgID, "UpdateDevice", httpResp, err)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to update device config %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// extractMerakiRadioBody pulls the radio settings to push from a config map, in the
// raw Meraki shape the endpoint accepts. It prefers an explicit radio_settings block
// (passed through verbatim — this is what the cache stores and apply diffs against),
// and falls back to translating an agnostic radio_config. The serial echoed back in a
// GET response and empty per-band blocks are dropped so a passthrough does not reset
// an unspecified band. Returns nil when there is nothing meaningful to send.
func extractMerakiRadioBody(config map[string]interface{}) map[string]any {
	if raw, ok := config["radio_settings"].(map[string]any); ok {
		return pruneRadioBody(raw)
	}
	if rc, ok := config["radio_config"]; ok {
		parsed := parseAgnosticRadioConfig(rc)
		if parsed == nil {
			return nil
		}
		return pruneRadioBody(vendors.NewRadioTranslator().ToMeraki(parsed))
	}
	return nil
}

// FilterApplicableRadio returns config keeping only the radio fields the Meraki
// integration can apply, plus the dotted names of what it dropped — derived by
// default-deny against the supported allowlist. Apply sends and compares only the
// applicable subset, so configuring a field the API cannot honor (e.g. 6 GHz) neither
// fails the apply nor reads as perpetual drift. The input map is not mutated. See #23.
func FilterApplicableRadio(config map[string]interface{}) (map[string]interface{}, []string) {
	rs, ok := config["radio_settings"].(map[string]any)
	if !ok {
		return config, nil
	}
	skipped := unsupportedRadioFields(rs)
	if len(skipped) == 0 {
		return config, nil
	}

	cleaned := make(map[string]any, len(rs))
	for k, v := range rs {
		cleaned[k] = v
	}
	for _, path := range skipped {
		band, sub, nested := strings.Cut(path, ".")
		if !nested {
			delete(cleaned, band)
			continue
		}
		if m, ok := cleaned[band].(map[string]any); ok {
			nm := make(map[string]any, len(m))
			for k, v := range m {
				if k != sub {
					nm[k] = v
				}
			}
			cleaned[band] = nm
		}
	}

	out := make(map[string]interface{}, len(config))
	for k, v := range config {
		out[k] = v
	}
	out["radio_settings"] = cleaned

	full := make([]string, len(skipped))
	for i, s := range skipped {
		full[i] = "radio_settings." + s
	}
	return out, full
}

// parseAgnosticRadioConfig converts a radio_config sub-map into the typed RadioConfig
// via a JSON round-trip, so the existing translator can render the Meraki body.
func parseAgnosticRadioConfig(rc any) *vendors.RadioConfig {
	data, err := json.Marshal(rc)
	if err != nil {
		logging.Warnf("[meraki] could not marshal radio_config: %v", err)
		return nil
	}
	var parsed vendors.RadioConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		logging.Warnf("[meraki] could not parse radio_config: %v", err)
		return nil
	}
	return &parsed
}

// pruneRadioBody copies a radio body, dropping the non-settable serial field and any
// empty per-band block so the PUT carries only fields the operator actually set.
func pruneRadioBody(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		if k == "serial" {
			continue
		}
		if m, ok := v.(map[string]any); ok && len(m) == 0 {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// supportedRadioBands allowlists, per band block, the radio fields the SDK's typed
// request can push. supportedRadioTopLevel does the same for top-level radio keys, and
// ignoredRadioFields are read-only echoes that are neither pushed nor reported. Anything
// outside these is unsupported by definition — default-deny, so a new or unmapped API
// field surfaces (and is skipped on push) instead of dropping silently.
// TODO(meraki-6ghz): widen support via a raw PUT to /devices/{serial}/wireless/radio/settings.
var (
	supportedRadioBands = map[string]map[string]bool{
		"twoFourGhzSettings": {"channel": true, "targetPower": true},
		"fiveGhzSettings":    {"channel": true, "channelWidth": true, "targetPower": true},
	}
	supportedRadioTopLevel = map[string]bool{"rfProfileId": true}
	ignoredRadioFields     = map[string]bool{"serial": true}
)

// unsupportedRadioFields returns the dotted names of radio-body fields the typed SDK
// request cannot push, derived by subtracting the supported allowlist — not an
// enumerated denylist. Unknown top-level blocks (e.g. sixGhzSettings) and unknown
// sub-fields of a known band (e.g. fiveGhzSettings.minBitrate) both surface. Sorted for
// stable output.
func unsupportedRadioFields(body map[string]any) []string {
	var out []string
	for key, val := range body {
		if supportedRadioTopLevel[key] || ignoredRadioFields[key] {
			continue
		}
		allowed, known := supportedRadioBands[key]
		if !known {
			out = append(out, key)
			continue
		}
		if band, ok := val.(map[string]any); ok {
			for sub := range band {
				if !allowed[sub] {
					out = append(out, key+"."+sub)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// applyRadioSettings pushes radio settings through the SDK's typed radio endpoint,
// sending only the allowlisted fields it can express (2.4/5 GHz manual settings and
// rfProfile). buildRadioRequest drops anything outside that set, so a config carrying
// fields the integration cannot apply (e.g. 6 GHz) sends the applicable subset rather
// than failing — the apply layer reports what it skipped.
func (s *devicesService) applyRadioSettings(ctx context.Context, deviceID string, body map[string]any) error {
	request := buildRadioRequest(body)
	if request == nil {
		return nil // no applicable radio fields to send
	}

	retryState := NewRetryState(s.retryConfig)
	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		httpResp, reqErr := s.dashboard.Wireless.UpdateDeviceWirelessRadioSettings(deviceID, request)
		err := ClassifyError(s.orgID, "UpdateDeviceWirelessRadioSettings", httpResp, reqErr)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to update radio settings for %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// buildRadioRequest maps the allowlisted fields of a radio body to the SDK's typed
// update request. Callers reject unsupported fields before reaching here; a nil result
// means the body holds nothing to send.
func buildRadioRequest(body map[string]any) *meraki.RequestWirelessUpdateDeviceWirelessRadioSettings {
	request := &meraki.RequestWirelessUpdateDeviceWirelessRadioSettings{}
	populated := false

	if id, ok := body["rfProfileId"].(string); ok && id != "" {
		request.RfProfileID = id
		populated = true
	}
	if tf, ok := body["twoFourGhzSettings"].(map[string]any); ok && len(tf) > 0 {
		request.TwoFourGhzSettings = &meraki.RequestWirelessUpdateDeviceWirelessRadioSettingsTwoFourGhzSettings{
			Channel:     toIntPtr(tf["channel"]),
			TargetPower: toIntPtr(tf["targetPower"]),
		}
		populated = true
	}
	if fv, ok := body["fiveGhzSettings"].(map[string]any); ok && len(fv) > 0 {
		request.FiveGhzSettings = &meraki.RequestWirelessUpdateDeviceWirelessRadioSettingsFiveGhzSettings{
			Channel:      toIntPtr(fv["channel"]),
			ChannelWidth: toIntPtr(fv["channelWidth"]),
			TargetPower:  toIntPtr(fv["targetPower"]),
		}
		populated = true
	}

	if !populated {
		return nil
	}
	return request
}

// toIntPtr coerces a JSON/map scalar to *int, tolerating the float64 from a JSON
// decode, the int from the in-process translator, and the string channelWidth the
// translator emits. Unparseable or absent values yield nil (the API reads that as auto).
func toIntPtr(v any) *int {
	switch n := v.(type) {
	case int:
		return &n
	case int64:
		x := int(n)
		return &x
	case float64:
		x := int(n)
		return &x
	case json.Number:
		if i, err := n.Int64(); err == nil {
			x := int(i)
			return &x
		}
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return &i
		}
	}
	return nil
}

// Reboot triggers an asynchronous reboot of a Meraki device. siteID is
// ignored — Meraki addresses devices by serial directly. Errors flow through
// ClassifyError so the caller gets the standard wifimgr-typed taxonomy
// (*AuthError, *RateLimitError, *ServerError, *NotFoundError, *TransportError).
//
// The SDK has historically panicked on flaky connections (e.g.
// `rand.Int63n(0)` inside its retry backoff). callRebootDevice wraps the
// SDK call in defer/recover so a panic surfaces as a retryable TransportError
// instead of taking the whole process down.
func (s *devicesService) Reboot(ctx context.Context, _, deviceID string) error {
	logging.Debugf("[meraki] Rebooting device serial=%s", deviceID)

	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		httpResp, err := s.callRebootDevice(deviceID)

		err = ClassifyError(s.orgID, "RebootDevice", httpResp, err)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to reboot device %s: %v", deviceID, err)
			return err
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// callRebootDevice invokes the SDK's RebootDevice with panic recovery. Any
// panic from the SDK (e.g. a math/rand panic from buggy backoff config under
// poor connectivity) is converted into a retryable transport-style error so
// ClassifyError can wrap it consistently with the rest of the taxonomy.
func (s *devicesService) callRebootDevice(deviceID string) (httpResp *resty.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Debugf("[meraki] SDK panic during RebootDevice(%s): %v", deviceID, r)
			httpResp = nil
			err = fmt.Errorf("meraki SDK panicked during reboot: %v", r)
		}
	}()

	if s.suppressOutput {
		restore := suppressStdout()
		defer restore()
	}
	_, httpResp, err = s.dashboard.Devices.RebootDevice(deviceID)
	return httpResp, err
}

// Ensure devicesService implements vendors.DevicesService at compile time.
var _ vendors.DevicesService = (*devicesService)(nil)
