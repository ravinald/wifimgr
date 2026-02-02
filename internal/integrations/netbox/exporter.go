package netbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Exporter orchestrates the export of devices to NetBox
type Exporter struct {
	client    *Client
	config    *Config
	validator *Validator
	mapper    *Mapper
	cache     *vendors.CacheAccessor
}

// NewExporter creates a new exporter
func NewExporter(config *Config) (*Exporter, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetBox client: %w", err)
	}

	validator := NewValidator(client, config)
	mapper := NewMapper(config, validator)

	cache := vendors.GetGlobalCacheAccessor()
	if cache == nil {
		return nil, fmt.Errorf("cache not initialized - run 'wifimgr cache refresh' first")
	}

	return &Exporter{
		client:    client,
		config:    config,
		validator: validator,
		mapper:    mapper,
		cache:     cache,
	}, nil
}

// Export performs the export operation based on the provided options
func (e *Exporter) Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	startTime := time.Now()

	result := &ExportResult{
		Created: make([]DeviceExportResult, 0),
		Updated: make([]DeviceExportResult, 0),
		Skipped: make([]SkippedDevice, 0),
		Errors:  make([]ExportError, 0),
	}

	// Initialize validator (fetch NetBox lookups)
	logging.Info("Initializing NetBox validator...")
	if err := e.validator.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	// Get devices to export
	devices, err := e.getDevicesToExport(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	result.Stats.TotalDevices = len(devices)
	logging.Infof("Found %d devices to export", len(devices))

	if len(devices) == 0 {
		result.Stats.Duration = time.Since(startTime).Round(time.Millisecond).String()
		return result, nil
	}

	// Process each device
	for _, device := range devices {
		e.processDevice(ctx, device, opts, result)
	}

	// Calculate final stats
	result.Stats.Created = len(result.Created)
	result.Stats.Updated = len(result.Updated)
	result.Stats.Skipped = len(result.Skipped)
	result.Stats.Errors = len(result.Errors)
	result.Stats.Duration = time.Since(startTime).Round(time.Millisecond).String()

	return result, nil
}

// getDevicesToExport retrieves devices based on export options.
// Only AP devices are exported; switches and gateways are excluded.
func (e *Exporter) getDevicesToExport(opts ExportOptions) ([]*vendors.InventoryItem, error) {
	var allDevices []*vendors.InventoryItem

	if opts.SiteName == "" {
		// Export all devices
		allDevices = e.cache.GetAllDevices()
	} else {
		// Export devices for a specific site
		site, err := e.cache.GetSiteByName(opts.SiteName)
		if err != nil {
			return nil, fmt.Errorf("site '%s' not found in wifimgr cache", opts.SiteName)
		}

		// Get all device types for this site
		allDevices = e.cache.GetDevicesBySite(site.ID, "")
	}

	// Filter to AP devices only
	var devices []*vendors.InventoryItem
	for _, device := range allDevices {
		if device.Type == "ap" {
			devices = append(devices, device)
		}
	}

	logging.Debugf("Filtered to %d AP devices (excluded %d non-AP devices)",
		len(devices), len(allDevices)-len(devices))

	// Resolve site names from site IDs when SiteName is empty
	// The cache stores site_id but site_name may be null
	e.enrichDevicesWithSiteNames(devices)

	return devices, nil
}

// enrichDevicesWithSiteNames resolves site names from site IDs for devices
// where SiteName is empty but SiteID is populated
func (e *Exporter) enrichDevicesWithSiteNames(devices []*vendors.InventoryItem) {
	for _, device := range devices {
		if device.SiteName == "" && device.SiteID != "" {
			if site, err := e.cache.GetSiteByID(device.SiteID); err == nil && site != nil {
				device.SiteName = site.Name
			}
		}
	}
}

// processDevice handles the export of a single device
func (e *Exporter) processDevice(ctx context.Context, item *vendors.InventoryItem, opts ExportOptions, result *ExportResult) {
	// Validate the device
	validation := e.validator.ValidateDevice(item)
	if !validation.Valid {
		result.Skipped = append(result.Skipped, SkippedDevice{
			Name:   item.Name,
			MAC:    item.MAC,
			Reason: fmt.Sprintf("validation failed: %v", validation.Errors),
		})
		return
	}

	// Check if device already exists in NetBox
	existingID, exists, err := e.validator.CheckDeviceExists(ctx, item.MAC)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "check_exists",
			Message:    "failed to check if device exists",
			Err:        err,
		})
		return
	}

	// Dry run - just report what would happen
	if opts.DryRun {
		if exists {
			result.Updated = append(result.Updated, DeviceExportResult{
				Name:      item.Name,
				MAC:       item.MAC,
				NetBoxID:  existingID,
				Operation: "would_update",
			})
		} else {
			result.Created = append(result.Created, DeviceExportResult{
				Name:      item.Name,
				MAC:       item.MAC,
				Operation: "would_create",
			})
		}
		return
	}

	// Create or update device
	if exists {
		e.updateDevice(ctx, item, existingID, validation, opts, result)
	} else {
		e.createDevice(ctx, item, validation, opts, result)
	}
}

// createDevice creates a new device in NetBox
func (e *Exporter) createDevice(ctx context.Context, item *vendors.InventoryItem, validation *DeviceValidationResult, opts ExportOptions, result *ExportResult) {
	// Map to NetBox request
	req, err := e.mapper.ToDeviceRequest(item, validation)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "map",
			Message:    "failed to map device",
			Err:        err,
		})
		return
	}

	// Create device in NetBox
	device, err := e.client.CreateDevice(ctx, req)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "create",
			Message:    "failed to create device",
			Err:        err,
		})
		return
	}

	logging.Debugf("Created device %s (ID: %d)", device.Name, device.ID)

	// Create primary interface with MAC address
	eth0, err := e.createInterfaceAndReturn(ctx, device.ID, item, result)
	if err != nil {
		// Interface creation failed, but device was created
		logging.Warnf("Device %s created but interface creation failed: %v", device.Name, err)
	} else if eth0 != nil {
		// Assign IP address to eth0 if available
		e.assignIPAddress(ctx, eth0.ID, item, result)
	}

	// Create radio interfaces if requested and device is an AP
	if opts.IncludeRadios && item.Type == "ap" {
		e.createRadioAndWLANInterfaces(ctx, device.ID, item, result)
	}

	result.Created = append(result.Created, DeviceExportResult{
		Name:      device.Name,
		MAC:       item.MAC,
		NetBoxID:  device.ID,
		Operation: "created",
	})
}

// updateDevice updates an existing device in NetBox
func (e *Exporter) updateDevice(ctx context.Context, item *vendors.InventoryItem, existingID int64, validation *DeviceValidationResult, opts ExportOptions, result *ExportResult) {
	// Map to NetBox request
	req, err := e.mapper.MapDeviceForUpdate(item, existingID, validation)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "map_update",
			Message:    "failed to map device for update",
			Err:        err,
		})
		return
	}

	// Update device in NetBox
	device, err := e.client.UpdateDevice(ctx, existingID, req)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "update",
			Message:    "failed to update device",
			Err:        err,
		})
		return
	}

	logging.Debugf("Updated device %s (ID: %d)", device.Name, device.ID)

	// Create radio interfaces if requested and device is an AP
	// Note: For updates, we only add radios if they don't exist yet
	if opts.IncludeRadios && item.Type == "ap" {
		e.createRadioAndWLANInterfaces(ctx, device.ID, item, result)
	}

	result.Updated = append(result.Updated, DeviceExportResult{
		Name:      device.Name,
		MAC:       item.MAC,
		NetBoxID:  device.ID,
		Operation: "updated",
	})
}

// createInterfaceAndReturn creates a primary interface for a device and returns it
func (e *Exporter) createInterfaceAndReturn(ctx context.Context, deviceID int64, item *vendors.InventoryItem, result *ExportResult) (*Interface, error) {
	ifaceReq, err := e.mapper.ToInterfaceRequest(deviceID, item)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "interface",
			Message:    "failed to map interface",
			Err:        err,
		})
		return nil, err
	}

	iface, err := e.client.CreateInterface(ctx, ifaceReq)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "interface",
			Message:    "failed to create interface",
			Err:        err,
		})
		return nil, err
	}

	logging.Debugf("Created interface %s for device (interface ID: %d)", iface.Name, iface.ID)
	return iface, nil
}

// assignIPAddress assigns a management IP address to an interface
func (e *Exporter) assignIPAddress(ctx context.Context, interfaceID int64, item *vendors.InventoryItem, result *ExportResult) {
	// Get device status from cache to find IP address
	status, err := e.cache.GetDeviceStatus(item.MAC)
	if err != nil || status == nil || status.IP == "" {
		// No IP available - device likely uses DHCP
		logging.Debugf("No IP address available for device %s", item.Name)
		return
	}

	// Ensure CIDR notation
	ip := status.IP
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}

	req := e.mapper.ToIPAddressRequest(interfaceID, ip)

	_, err = e.client.CreateIPAddress(ctx, req)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "ip_address",
			Message:    "failed to assign IP address",
			Err:        err,
		})
		return
	}

	logging.Debugf("Assigned IP %s to device %s", ip, item.Name)
}

// createRadioAndWLANInterfaces creates radio interfaces and WLAN virtual interfaces for an AP
func (e *Exporter) createRadioAndWLANInterfaces(ctx context.Context, deviceID int64, item *vendors.InventoryItem, result *ExportResult) {
	// Get AP config from cache to extract radio configuration
	apConfig, err := e.cache.GetAPConfigByMAC(item.MAC)
	if err != nil || apConfig == nil {
		logging.Debugf("No AP config found for %s, skipping radio interfaces", item.Name)
		return
	}

	// Parse radio configuration from the raw config
	radioConfig := e.parseRadioConfig(apConfig.Config)
	if radioConfig == nil {
		logging.Debugf("No radio config found in AP config for %s", item.Name)
		return
	}

	// Create physical radio interfaces (wifi0, wifi1, wifi2)
	radioRequests, err := e.mapper.ToRadioInterfaceRequests(deviceID, radioConfig, item)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "radio_interfaces",
			Message:    "failed to map radio interfaces",
			Err:        err,
		})
		return
	}
	if len(radioRequests) == 0 {
		logging.Debugf("No radio interfaces to create for %s", item.Name)
		return
	}

	radioInterfaces, err := e.client.BulkCreateInterfaces(ctx, radioRequests)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "radio_interfaces",
			Message:    "failed to create radio interfaces",
			Err:        err,
		})
		return
	}

	logging.Debugf("Created %d radio interfaces for device %s", len(radioInterfaces), item.Name)

	// Build map of radio name -> interface ID
	radioMap := make(map[string]int64)
	for _, iface := range radioInterfaces {
		radioMap[iface.Name] = iface.ID
	}

	// Get WLANs for this site to create virtual interfaces
	wlans := e.getWLANsForDevice(item)
	if len(wlans) == 0 {
		logging.Debugf("No WLANs found for device %s", item.Name)
		return
	}

	// Ensure WirelessLAN objects exist in NetBox and get their IDs
	wirelessLANIDs := e.ensureWirelessLANs(ctx, wlans, result)

	// Create virtual WLAN interfaces linked to radios
	virtualRequests := e.mapper.ToVirtualWLANInterfaceRequests(deviceID, radioMap, wlans, wirelessLANIDs)
	if len(virtualRequests) == 0 {
		return
	}

	virtualInterfaces, err := e.client.BulkCreateInterfaces(ctx, virtualRequests)
	if err != nil {
		result.Errors = append(result.Errors, ExportError{
			DeviceName: item.Name,
			DeviceMAC:  item.MAC,
			Operation:  "wlan_interfaces",
			Message:    "failed to create WLAN virtual interfaces",
			Err:        err,
		})
		return
	}

	logging.Debugf("Created %d WLAN virtual interfaces for device %s", len(virtualInterfaces), item.Name)
}

// parseRadioConfig extracts RadioConfig from raw AP config
func (e *Exporter) parseRadioConfig(config map[string]any) *vendors.RadioConfig {
	if config == nil {
		return nil
	}

	radioConfig := &vendors.RadioConfig{}
	hasAnyBand := false

	// Look for radio_config in the raw config (Mist format)
	if rc, ok := config["radio_config"].(map[string]any); ok {
		if band24, ok := rc["band_24"].(map[string]any); ok {
			radioConfig.Band24 = e.parseBandConfig(band24)
			hasAnyBand = true
		}
		if band5, ok := rc["band_5"].(map[string]any); ok {
			radioConfig.Band5 = e.parseBandConfig(band5)
			hasAnyBand = true
		}
		if band6, ok := rc["band_6"].(map[string]any); ok {
			radioConfig.Band6 = e.parseBandConfig(band6)
			hasAnyBand = true
		}
	}

	// If no explicit radio config, assume default dual-band
	if !hasAnyBand {
		radioConfig.Band24 = &vendors.RadioBandConfig{}
		radioConfig.Band5 = &vendors.RadioBandConfig{}
		hasAnyBand = true
	}

	if hasAnyBand {
		return radioConfig
	}
	return nil
}

// parseBandConfig parses a single band configuration
func (e *Exporter) parseBandConfig(config map[string]any) *vendors.RadioBandConfig {
	bandConfig := &vendors.RadioBandConfig{}

	if disabled, ok := config["disabled"].(bool); ok {
		bandConfig.Disabled = &disabled
	}
	if channel, ok := config["channel"].(float64); ok {
		ch := int(channel)
		bandConfig.Channel = &ch
	}
	if power, ok := config["power"].(float64); ok {
		p := int(power)
		bandConfig.Power = &p
	}

	return bandConfig
}

// getWLANsForDevice returns WLANs applicable to a device based on its site
func (e *Exporter) getWLANsForDevice(item *vendors.InventoryItem) []*vendors.WLAN {
	if item.SiteID == "" {
		return nil
	}

	// Get WLANs for this site from cache
	return e.cache.GetWLANsBySite(item.SiteID)
}

// ensureWirelessLANs creates WirelessLAN objects in NetBox if they don't exist
func (e *Exporter) ensureWirelessLANs(ctx context.Context, wlans []*vendors.WLAN, _ *ExportResult) map[string]int64 {
	ssidToID := make(map[string]int64)
	var toCreate []*WirelessLANRequest

	for _, wlan := range wlans {
		// Check if already exists in cache
		if id, ok := e.validator.GetWirelessLANID(wlan.SSID); ok {
			ssidToID[wlan.SSID] = id
			continue
		}

		// Check if we already have it in our toCreate list (dedup)
		if _, exists := ssidToID[wlan.SSID]; exists {
			continue
		}

		toCreate = append(toCreate, e.mapper.ToWirelessLANRequest(wlan))
		ssidToID[wlan.SSID] = 0 // Placeholder to mark as pending
	}

	if len(toCreate) == 0 {
		return ssidToID
	}

	// Create missing WirelessLANs
	created, err := e.client.BulkCreateWirelessLANs(ctx, toCreate)
	if err != nil {
		logging.Warnf("Failed to create WirelessLANs: %v", err)
		return ssidToID
	}

	// Update map with created IDs
	for _, wl := range created {
		ssidToID[wl.SSID] = wl.ID
		logging.Debugf("Created WirelessLAN %s (ID: %d)", wl.SSID, wl.ID)
	}

	return ssidToID
}

// GetValidationSummary returns a summary of which NetBox dependencies are available
func (e *Exporter) GetValidationSummary() map[string]int {
	return e.validator.GetCacheStats()
}

// ValidateOnly performs validation without exporting
func (e *Exporter) ValidateOnly(ctx context.Context, opts ExportOptions) (*ValidationSummary, error) {
	// Initialize validator
	if err := e.validator.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	// Get devices
	devices, err := e.getDevicesToExport(opts)
	if err != nil {
		return nil, err
	}

	summary := &ValidationSummary{
		TotalDevices:     len(devices),
		ValidDevices:     0,
		InvalidDevices:   0,
		MissingSites:     make(map[string]int),
		MissingTypes:     make(map[string]int),
		MissingRoles:     make(map[string]int),
		ValidationErrors: make([]DeviceValidationSummary, 0),
	}

	for _, device := range devices {
		validation := e.validator.ValidateDevice(device)
		if validation.Valid {
			summary.ValidDevices++
		} else {
			summary.InvalidDevices++
			summary.ValidationErrors = append(summary.ValidationErrors, DeviceValidationSummary{
				Name:   device.Name,
				MAC:    device.MAC,
				Errors: validation.Errors,
			})

			// Track missing dependencies
			for _, errMsg := range validation.Errors {
				if contains(errMsg, "site") {
					summary.MissingSites[device.SiteName]++
				}
				if contains(errMsg, "device type") {
					summary.MissingTypes[device.Model]++
				}
				if contains(errMsg, "device role") {
					summary.MissingRoles[device.Type]++
				}
			}
		}
	}

	return summary, nil
}

// ValidationSummary contains summary of validation results
type ValidationSummary struct {
	TotalDevices     int
	ValidDevices     int
	InvalidDevices   int
	MissingSites     map[string]int // site name -> count of devices
	MissingTypes     map[string]int // model -> count of devices
	MissingRoles     map[string]int // device type -> count of devices
	ValidationErrors []DeviceValidationSummary
}

// DeviceValidationSummary contains validation errors for a single device
type DeviceValidationSummary struct {
	Name   string
	MAC    string
	Errors []string
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsCI(s, substr))
}

func containsCI(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if eqCI(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func eqCI(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
