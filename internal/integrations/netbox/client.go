package netbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/netbox-community/go-netbox/v4"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// BulkBatchSize is the maximum number of items to create/delete in a single bulk API call
const BulkBatchSize = 100

// Client wraps the go-netbox API client with convenience methods
type Client struct {
	api    *netbox.APIClient
	config *Config
}

// NewClient creates a new NetBox client from configuration
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create base HTTP client with SSL configuration
	httpClient := &http.Client{}
	if !cfg.SSLVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		logging.Warnf("NetBox SSL verification is disabled")
	}

	// Normalize URL (remove trailing slash)
	url := strings.TrimSuffix(cfg.URL, "/")

	// Create netbox client configuration
	apiConfig := netbox.NewConfiguration()
	apiConfig.Servers = netbox.ServerConfigurations{
		{URL: url},
	}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Token %s", cfg.GetAPIKey()))
	apiConfig.HTTPClient = httpClient

	client := netbox.NewAPIClient(apiConfig)

	return &Client{
		api:    client,
		config: cfg,
	}, nil
}

// GetSiteByName finds a NetBox site by name
func (c *Client) GetSiteByName(ctx context.Context, name string) (*Site, error) {
	res, _, err := c.api.DcimAPI.DcimSitesList(ctx).
		Name([]string{name}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil // Not found
	}

	site := res.Results[0]
	status := ""
	if site.Status != nil && site.Status.Value != nil {
		status = string(*site.Status.Value)
	}
	return &Site{
		ID:     int64(site.Id),
		Name:   site.Name,
		Slug:   site.Slug,
		Status: status,
	}, nil
}

// GetSiteBySlug finds a NetBox site by slug
func (c *Client) GetSiteBySlug(ctx context.Context, slug string) (*Site, error) {
	res, _, err := c.api.DcimAPI.DcimSitesList(ctx).
		Slug([]string{slug}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	site := res.Results[0]
	status := ""
	if site.Status != nil && site.Status.Value != nil {
		status = string(*site.Status.Value)
	}
	return &Site{
		ID:     int64(site.Id),
		Name:   site.Name,
		Slug:   site.Slug,
		Status: status,
	}, nil
}

// GetAllSites returns all NetBox sites
func (c *Client) GetAllSites(ctx context.Context) ([]*Site, error) {
	var sites []*Site
	limit := int32(100)
	offset := int32(0)

	for {
		res, _, err := c.api.DcimAPI.DcimSitesList(ctx).
			Limit(limit).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to query sites: %w", err)
		}

		for _, site := range res.Results {
			status := ""
			if site.Status != nil && site.Status.Value != nil {
				status = string(*site.Status.Value)
			}
			sites = append(sites, &Site{
				ID:     int64(site.Id),
				Name:   site.Name,
				Slug:   site.Slug,
				Status: status,
			})
		}

		if res.Next.Get() == nil || *res.Next.Get() == "" {
			break
		}
		offset += limit
	}

	return sites, nil
}

// GetDeviceTypeBySlug finds a NetBox device type by slug
func (c *Client) GetDeviceTypeBySlug(ctx context.Context, slug string) (*DeviceType, error) {
	res, _, err := c.api.DcimAPI.DcimDeviceTypesList(ctx).
		Slug([]string{slug}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query device types: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	dt := res.Results[0]
	return &DeviceType{
		ID:           int64(dt.Id),
		Manufacturer: dt.Manufacturer.Name,
		Model:        dt.Model,
		Slug:         dt.Slug,
	}, nil
}

// GetAllDeviceTypes returns all NetBox device types
func (c *Client) GetAllDeviceTypes(ctx context.Context) ([]*DeviceType, error) {
	var deviceTypes []*DeviceType
	limit := int32(100)
	offset := int32(0)

	for {
		res, _, err := c.api.DcimAPI.DcimDeviceTypesList(ctx).
			Limit(limit).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to query device types: %w", err)
		}

		for _, dt := range res.Results {
			deviceTypes = append(deviceTypes, &DeviceType{
				ID:           int64(dt.Id),
				Manufacturer: dt.Manufacturer.Name,
				Model:        dt.Model,
				Slug:         dt.Slug,
			})
		}

		if res.Next.Get() == nil || *res.Next.Get() == "" {
			break
		}
		offset += limit
	}

	return deviceTypes, nil
}

// GetDeviceRoleBySlug finds a NetBox device role by slug
func (c *Client) GetDeviceRoleBySlug(ctx context.Context, slug string) (*DeviceRole, error) {
	res, _, err := c.api.DcimAPI.DcimDeviceRolesList(ctx).
		Slug([]string{slug}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query device roles: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	role := res.Results[0]
	return &DeviceRole{
		ID:   int64(role.Id),
		Name: role.Name,
		Slug: role.Slug,
	}, nil
}

// GetAllDeviceRoles returns all NetBox device roles
func (c *Client) GetAllDeviceRoles(ctx context.Context) ([]*DeviceRole, error) {
	var roles []*DeviceRole
	limit := int32(100)
	offset := int32(0)

	for {
		res, _, err := c.api.DcimAPI.DcimDeviceRolesList(ctx).
			Limit(limit).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to query device roles: %w", err)
		}

		for _, role := range res.Results {
			roles = append(roles, &DeviceRole{
				ID:   int64(role.Id),
				Name: role.Name,
				Slug: role.Slug,
			})
		}

		if res.Next.Get() == nil || *res.Next.Get() == "" {
			break
		}
		offset += limit
	}

	return roles, nil
}

// GetDeviceByMAC finds a NetBox device by interface MAC address
func (c *Client) GetDeviceByMAC(ctx context.Context, mac string) (*Device, error) {
	// Normalize MAC address to NetBox format (XX:XX:XX:XX:XX:XX)
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return nil, fmt.Errorf("invalid MAC address: %s", mac)
	}

	// Convert to colon-separated format for NetBox
	netboxMAC := formatMACForNetBox(normalizedMAC)

	// First, find the interface with this MAC
	res, _, err := c.api.DcimAPI.DcimInterfacesList(ctx).
		MacAddress([]string{netboxMAC}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query interfaces: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil // Not found
	}

	iface := res.Results[0]
	deviceID := int64(iface.Device.Id)

	// Now get the full device
	return c.GetDeviceByID(ctx, deviceID)
}

// GetDeviceByID retrieves a device by its NetBox ID
func (c *Client) GetDeviceByID(ctx context.Context, id int64) (*Device, error) {
	device, _, err := c.api.DcimAPI.DcimDevicesRetrieve(ctx, int32(id)).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return convertDevice(device), nil
}

// GetDeviceByName finds a NetBox device by name
func (c *Client) GetDeviceByName(ctx context.Context, name string) (*Device, error) {
	res, _, err := c.api.DcimAPI.DcimDevicesList(ctx).
		Name([]string{name}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	return convertDeviceWithType(&res.Results[0]), nil
}

// GetDevicesBySiteAndRole retrieves all devices from NetBox filtered by site and role
func (c *Client) GetDevicesBySiteAndRole(ctx context.Context, siteSlug string, roleSlug string) ([]*Device, error) {
	var devices []*Device
	limit := int32(100)
	offset := int32(0)

	for {
		req := c.api.DcimAPI.DcimDevicesList(ctx).
			Limit(limit).
			Offset(offset)

		// Apply filters
		if siteSlug != "" {
			req = req.Site([]string{siteSlug})
		}
		if roleSlug != "" {
			req = req.Role([]string{roleSlug})
		}

		res, _, err := req.Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to query devices: %w", err)
		}

		for i := range res.Results {
			devices = append(devices, convertDeviceWithType(&res.Results[i]))
		}

		if res.Next.Get() == nil || *res.Next.Get() == "" {
			break
		}
		offset += limit
	}

	return devices, nil
}

// CreateDevice creates a new device in NetBox
func (c *Client) CreateDevice(ctx context.Context, req *DeviceRequest) (*Device, error) {
	// Convert IDs to the union types expected by the API
	deviceTypeID := int32(req.DeviceType)
	roleID := int32(req.Role)
	siteID := int32(req.Site)

	deviceReq := netbox.WritableDeviceWithConfigContextRequest{
		Name:       *netbox.NewNullableString(&req.Name),
		DeviceType: netbox.Int32AsDeviceBayTemplateRequestDeviceType(&deviceTypeID),
		Role:       netbox.Int32AsDeviceWithConfigContextRequestRole(&roleID),
		Site:       netbox.Int32AsDeviceWithConfigContextRequestSite(&siteID),
	}

	if req.Serial != "" {
		deviceReq.Serial = &req.Serial
	}
	if req.Status != "" {
		status := netbox.DeviceStatusValue(req.Status)
		deviceReq.Status = &status
	}
	if req.Comments != "" {
		deviceReq.Comments = &req.Comments
	}

	device, _, err := c.api.DcimAPI.DcimDevicesCreate(ctx).
		WritableDeviceWithConfigContextRequest(deviceReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return convertDevice(device), nil
}

// UpdateDevice updates an existing device in NetBox
func (c *Client) UpdateDevice(ctx context.Context, id int64, req *DeviceRequest) (*Device, error) {
	// Convert IDs to the union types expected by the API
	deviceTypeID := int32(req.DeviceType)
	roleID := int32(req.Role)
	siteID := int32(req.Site)

	deviceReq := netbox.WritableDeviceWithConfigContextRequest{
		Name:       *netbox.NewNullableString(&req.Name),
		DeviceType: netbox.Int32AsDeviceBayTemplateRequestDeviceType(&deviceTypeID),
		Role:       netbox.Int32AsDeviceWithConfigContextRequestRole(&roleID),
		Site:       netbox.Int32AsDeviceWithConfigContextRequestSite(&siteID),
	}

	if req.Serial != "" {
		deviceReq.Serial = &req.Serial
	}
	if req.Status != "" {
		status := netbox.DeviceStatusValue(req.Status)
		deviceReq.Status = &status
	}
	if req.Comments != "" {
		deviceReq.Comments = &req.Comments
	}

	device, _, err := c.api.DcimAPI.DcimDevicesUpdate(ctx, int32(id)).
		WritableDeviceWithConfigContextRequest(deviceReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return convertDevice(device), nil
}

// CreateInterface creates a new interface on a device
func (c *Client) CreateInterface(ctx context.Context, req *InterfaceRequest) (*Interface, error) {
	deviceID := int32(req.Device)

	ifaceReq := netbox.WritableInterfaceRequest{
		Device: netbox.Int32AsBriefInterfaceRequestDevice(&deviceID),
		Name:   req.Name,
		Type:   netbox.InterfaceTypeValue(req.Type),
	}

	// Note: MAC address is set via PrimaryMacAddress or through a separate interface MAC endpoint
	// The WritableInterfaceRequest doesn't have a direct MacAddress field in this version

	iface, _, err := c.api.DcimAPI.DcimInterfacesCreate(ctx).
		WritableInterfaceRequest(ifaceReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	return convertInterface(iface), nil
}

// GetInterfacesByDevice returns all interfaces for a device
func (c *Client) GetInterfacesByDevice(ctx context.Context, deviceID int64) ([]*Interface, error) {
	res, _, err := c.api.DcimAPI.DcimInterfacesList(ctx).
		DeviceId([]int32{int32(deviceID)}).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query interfaces: %w", err)
	}

	var ifaces []*Interface
	for i := range res.Results {
		ifaces = append(ifaces, convertInterface(&res.Results[i]))
	}

	return ifaces, nil
}

// UpdateInterface updates an existing interface
func (c *Client) UpdateInterface(ctx context.Context, id int64, req *InterfaceUpdateRequest) (*Interface, error) {
	// Fetch the existing interface to get required fields
	existing, _, err := c.api.DcimAPI.DcimInterfacesRetrieve(ctx, int32(id)).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve interface: %w", err)
	}

	// Build patch request with only the fields we want to update
	deviceID := existing.Device.Id
	deviceRef := netbox.Int32AsBriefInterfaceRequestDevice(&deviceID)
	patchReq := netbox.PatchedWritableInterfaceRequest{
		Device: &deviceRef,
		Name:   &existing.Name,
	}

	if req.Type != "" {
		ifaceType := netbox.InterfaceTypeValue(req.Type)
		patchReq.Type = &ifaceType
	}
	if req.Enabled != nil {
		patchReq.Enabled = req.Enabled
	}

	iface, _, err := c.api.DcimAPI.DcimInterfacesPartialUpdate(ctx, int32(id)).
		PatchedWritableInterfaceRequest(patchReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update interface: %w", err)
	}

	return convertInterface(iface), nil
}

// GetInterfaceTemplates returns all interface templates for a device type
func (c *Client) GetInterfaceTemplates(ctx context.Context, deviceTypeID int64) ([]*InterfaceTemplate, error) {
	dtID := int32(deviceTypeID)
	res, _, err := c.api.DcimAPI.DcimInterfaceTemplatesList(ctx).
		DeviceTypeId([]*int32{&dtID}).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query interface templates: %w", err)
	}

	var templates []*InterfaceTemplate
	for _, tmpl := range res.Results {
		ifaceType := ""
		if tmpl.Type.Value != nil {
			ifaceType = string(*tmpl.Type.Value)
		}
		templates = append(templates, &InterfaceTemplate{
			ID:   int64(tmpl.Id),
			Name: tmpl.Name,
			Type: ifaceType,
		})
	}

	return templates, nil
}

// CreateIPAddress creates a new IP address in NetBox
func (c *Client) CreateIPAddress(ctx context.Context, req *IPAddressRequest) (*IPAddress, error) {
	ipReq := netbox.WritableIPAddressRequest{
		Address: req.Address,
	}

	if req.Status != "" {
		status := netbox.PatchedWritableIPAddressRequestStatus(req.Status)
		ipReq.Status = &status
	}
	if req.AssignedObjectType != "" && req.AssignedObjectID > 0 {
		ipReq.AssignedObjectType = *netbox.NewNullableString(&req.AssignedObjectType)
		ipReq.AssignedObjectId = *netbox.NewNullableInt64(&req.AssignedObjectID)
	}
	if req.DNSName != "" {
		ipReq.DnsName = &req.DNSName
	}

	ip, _, err := c.api.IpamAPI.IpamIpAddressesCreate(ctx).
		WritableIPAddressRequest(ipReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create IP address: %w", err)
	}

	return convertIPAddress(ip), nil
}

// GetIPAddressByAddress finds an IP address by its address string
func (c *Client) GetIPAddressByAddress(ctx context.Context, address string) (*IPAddress, error) {
	res, _, err := c.api.IpamAPI.IpamIpAddressesList(ctx).
		Address([]string{address}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query IP addresses: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	ip := res.Results[0]
	return convertIPAddress(&ip), nil
}

// UpdateIPAddress updates an existing IP address
func (c *Client) UpdateIPAddress(ctx context.Context, id int64, req *IPAddressRequest) (*IPAddress, error) {
	ipReq := netbox.WritableIPAddressRequest{
		Address: req.Address,
	}

	if req.Status != "" {
		status := netbox.PatchedWritableIPAddressRequestStatus(req.Status)
		ipReq.Status = &status
	}
	if req.AssignedObjectType != "" && req.AssignedObjectID > 0 {
		ipReq.AssignedObjectType = *netbox.NewNullableString(&req.AssignedObjectType)
		ipReq.AssignedObjectId = *netbox.NewNullableInt64(&req.AssignedObjectID)
	}
	if req.DNSName != "" {
		ipReq.DnsName = &req.DNSName
	}

	ip, _, err := c.api.IpamAPI.IpamIpAddressesUpdate(ctx, int32(id)).
		WritableIPAddressRequest(ipReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update IP address: %w", err)
	}

	return convertIPAddress(ip), nil
}

// TestConnection tests the connection to NetBox
func (c *Client) TestConnection(ctx context.Context) error {
	_, _, err := c.api.StatusAPI.StatusRetrieve(ctx).Execute()
	if err != nil {
		return fmt.Errorf("failed to connect to NetBox: %w", err)
	}
	return nil
}

// Helper functions

func convertDevice(d *netbox.DeviceWithConfigContext) *Device {
	device := &Device{
		ID:     int64(d.Id),
		Status: "",
	}

	if d.Name.Get() != nil {
		device.Name = *d.Name.Get()
	}

	if d.Status != nil && d.Status.Value != nil {
		device.Status = string(*d.Status.Value)
	}

	if d.DeviceType.Id != 0 {
		device.DeviceType = &DeviceType{
			ID:           int64(d.DeviceType.Id),
			Model:        d.DeviceType.Model,
			Slug:         d.DeviceType.Slug,
			Manufacturer: d.DeviceType.Manufacturer.Name,
		}
	}

	if d.Role.Id != 0 {
		device.Role = &DeviceRole{
			ID:   int64(d.Role.Id),
			Name: d.Role.Name,
			Slug: d.Role.Slug,
		}
	}

	if d.Site.Id != 0 {
		device.Site = &Site{
			ID:   int64(d.Site.Id),
			Name: d.Site.Name,
			Slug: d.Site.Slug,
		}
	}

	if d.Serial != nil {
		device.Serial = *d.Serial
	}

	if d.Comments != nil {
		device.Comments = *d.Comments
	}

	// Handle nullable IP addresses
	if d.PrimaryIp4.IsSet() && d.PrimaryIp4.Get() != nil {
		device.PrimaryIPv4 = d.PrimaryIp4.Get().Address
	}
	if d.PrimaryIp6.IsSet() && d.PrimaryIp6.Get() != nil {
		device.PrimaryIPv6 = d.PrimaryIp6.Get().Address
	}
	if d.PrimaryIp.IsSet() && d.PrimaryIp.Get() != nil {
		device.PrimaryIP = d.PrimaryIp.Get().Address
	}

	return device
}

func convertDeviceWithType(d *netbox.DeviceWithConfigContext) *Device {
	return convertDevice(d)
}

func convertInterface(iface *netbox.Interface) *Interface {
	result := &Interface{
		ID:      int64(iface.Id),
		Device:  int64(iface.Device.Id),
		Name:    iface.Name,
		Enabled: true,
	}

	if iface.Type.Value != nil {
		result.Type = string(*iface.Type.Value)
	}

	if iface.MacAddress.Get() != nil {
		result.MACAddr = *iface.MacAddress.Get()
	}

	if iface.Enabled != nil {
		result.Enabled = *iface.Enabled
	}

	return result
}

func convertIPAddress(ip *netbox.IPAddress) *IPAddress {
	result := &IPAddress{
		ID:      int64(ip.Id),
		Address: ip.Address,
	}

	if ip.Status != nil && ip.Status.Value != nil {
		result.Status = string(*ip.Status.Value)
	}

	if ip.DnsName != nil {
		result.DNSName = *ip.DnsName
	}

	return result
}

// formatMACForNetBox converts a MAC address to NetBox format (XX:XX:XX:XX:XX:XX)
func formatMACForNetBox(mac string) string {
	// Remove all separators
	clean := strings.ReplaceAll(mac, ":", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, ".", "")
	clean = strings.ToUpper(clean)

	if len(clean) != 12 {
		return mac // Return original if invalid
	}

	// Format as XX:XX:XX:XX:XX:XX
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		clean[0:2], clean[2:4], clean[4:6],
		clean[6:8], clean[8:10], clean[10:12])
}

// BulkCreateInterfaces creates multiple interfaces in batches
func (c *Client) BulkCreateInterfaces(ctx context.Context, reqs []*InterfaceRequest) ([]*Interface, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	var results []*Interface
	batches := splitIntoBatches(reqs, BulkBatchSize)

	for _, batch := range batches {
		created, err := c.bulkCreateInterfaces(ctx, batch)
		if err != nil {
			return results, err
		}
		results = append(results, created...)
	}

	return results, nil
}

// bulkCreateInterfaces creates a batch of interfaces via bulk API
func (c *Client) bulkCreateInterfaces(ctx context.Context, reqs []*InterfaceRequest) ([]*Interface, error) {
	url := fmt.Sprintf("%s/api/dcim/interfaces/", strings.TrimSuffix(c.config.URL, "/"))

	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal interface requests: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.config.GetAPIKey()))
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.api.GetConfig().HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bulk create: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bulk create failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var results []*Interface
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf("failed to parse bulk create response: %w", err)
	}

	return results, nil
}

// BulkDeleteInterfaces deletes multiple interfaces by ID
func (c *Client) BulkDeleteInterfaces(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	batches := splitIntoBatchesInt64(ids, BulkBatchSize)
	for _, batch := range batches {
		if err := c.bulkDeleteInterfaces(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

// bulkDeleteInterfaces deletes a batch of interfaces via bulk API
func (c *Client) bulkDeleteInterfaces(ctx context.Context, ids []int64) error {
	url := fmt.Sprintf("%s/api/dcim/interfaces/", strings.TrimSuffix(c.config.URL, "/"))

	deleteReqs := make([]map[string]int64, len(ids))
	for i, id := range ids {
		deleteReqs[i] = map[string]int64{"id": id}
	}

	body, err := json.Marshal(deleteReqs)
	if err != nil {
		return fmt.Errorf("failed to marshal delete requests: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.config.GetAPIKey()))
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.api.GetConfig().HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute bulk delete: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk delete failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetAllWirelessLANs returns all WirelessLAN objects from NetBox
func (c *Client) GetAllWirelessLANs(ctx context.Context) ([]*WirelessLAN, error) {
	var wirelessLANs []*WirelessLAN
	limit := int32(100)
	offset := int32(0)

	for {
		res, _, err := c.api.WirelessAPI.WirelessWirelessLansList(ctx).
			Limit(limit).
			Offset(offset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to query wireless LANs: %w", err)
		}

		for _, wlan := range res.Results {
			wirelessLANs = append(wirelessLANs, convertWirelessLAN(&wlan))
		}

		if res.Next.Get() == nil || *res.Next.Get() == "" {
			break
		}
		offset += limit
	}

	return wirelessLANs, nil
}

// GetWirelessLANBySSID finds a WirelessLAN by SSID
func (c *Client) GetWirelessLANBySSID(ctx context.Context, ssid string) (*WirelessLAN, error) {
	res, _, err := c.api.WirelessAPI.WirelessWirelessLansList(ctx).
		Ssid([]string{ssid}).
		Limit(1).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query wireless LANs: %w", err)
	}

	if res.Count == 0 || len(res.Results) == 0 {
		return nil, nil
	}

	return convertWirelessLAN(&res.Results[0]), nil
}

// CreateWirelessLAN creates a new WirelessLAN in NetBox
func (c *Client) CreateWirelessLAN(ctx context.Context, req *WirelessLANRequest) (*WirelessLAN, error) {
	wlanReq := netbox.WritableWirelessLANRequest{
		Ssid: req.SSID,
	}

	if req.Status != "" {
		status := netbox.PatchedWritableWirelessLANRequestStatus(req.Status)
		wlanReq.Status = &status
	}
	if req.AuthType != "" {
		authType := netbox.AuthenticationType1(req.AuthType)
		wlanReq.AuthType = *netbox.NewNullableAuthenticationType1(&authType)
	}
	if req.AuthCipher != "" {
		authCipher := netbox.AuthenticationCipher(req.AuthCipher)
		wlanReq.AuthCipher = *netbox.NewNullableAuthenticationCipher(&authCipher)
	}

	wlan, _, err := c.api.WirelessAPI.WirelessWirelessLansCreate(ctx).
		WritableWirelessLANRequest(wlanReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create wireless LAN: %w", err)
	}

	return convertWirelessLAN(wlan), nil
}

// BulkCreateWirelessLANs creates multiple WirelessLANs in batches
func (c *Client) BulkCreateWirelessLANs(ctx context.Context, reqs []*WirelessLANRequest) ([]*WirelessLAN, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	var results []*WirelessLAN
	batches := splitIntoBatchesWLAN(reqs, BulkBatchSize)

	for _, batch := range batches {
		created, err := c.bulkCreateWirelessLANs(ctx, batch)
		if err != nil {
			return results, err
		}
		results = append(results, created...)
	}

	return results, nil
}

// bulkCreateWirelessLANs creates a batch of WirelessLANs via bulk API
func (c *Client) bulkCreateWirelessLANs(ctx context.Context, reqs []*WirelessLANRequest) ([]*WirelessLAN, error) {
	url := fmt.Sprintf("%s/api/wireless/wireless-lans/", strings.TrimSuffix(c.config.URL, "/"))

	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wireless LAN requests: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.config.GetAPIKey()))
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.api.GetConfig().HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bulk create: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bulk create wireless LANs failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var results []*WirelessLAN
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf("failed to parse bulk create response: %w", err)
	}

	return results, nil
}

// convertWirelessLAN converts a NetBox API WirelessLAN to our type
func convertWirelessLAN(wlan *netbox.WirelessLAN) *WirelessLAN {
	result := &WirelessLAN{
		ID:   int64(wlan.Id),
		SSID: wlan.Ssid,
	}

	if wlan.Status != nil && wlan.Status.Value != nil {
		result.Status = string(*wlan.Status.Value)
	}
	if wlan.AuthType != nil && wlan.AuthType.Value != nil {
		result.AuthType = string(*wlan.AuthType.Value)
	}
	if wlan.AuthCipher != nil && wlan.AuthCipher.Value != nil {
		result.AuthCipher = string(*wlan.AuthCipher.Value)
	}

	return result
}

// splitIntoBatches splits a slice of pointers into batches
func splitIntoBatches[T any](items []*T, batchSize int) [][]*T {
	var batches [][]*T
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batches = append(batches, items[i:end])
	}
	return batches
}

// splitIntoBatchesInt64 splits a slice of int64 into batches
func splitIntoBatchesInt64(items []int64, batchSize int) [][]int64 {
	var batches [][]int64
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batches = append(batches, items[i:end])
	}
	return batches
}

// splitIntoBatchesWLAN splits a slice of WirelessLANRequests into batches
func splitIntoBatchesWLAN(items []*WirelessLANRequest, batchSize int) [][]*WirelessLANRequest {
	var batches [][]*WirelessLANRequest
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batches = append(batches, items[i:end])
	}
	return batches
}
