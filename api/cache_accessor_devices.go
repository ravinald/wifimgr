package api

import (
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// GetAPByMAC retrieves an AP by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetAPByMAC(mac string) (*APDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	ap, exists := indexes.APsByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("AP not found: %s", mac)
	}

	return ap, nil
}

// GetAPByName retrieves an AP by name with O(1) lookup
func (ca *CacheAccessorImpl) GetAPByName(name string) (*APDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	ap, exists := indexes.APsByName[name]
	if !exists {
		return nil, fmt.Errorf("AP not found: %s", name)
	}

	return ap, nil
}

// GetAPsBySite retrieves all APs for a specific site with O(1) lookup
func (ca *CacheAccessorImpl) GetAPsBySite(siteID string) ([]*APDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	aps, exists := indexes.APsBySite[siteID]
	if !exists {
		return []*APDevice{}, nil
	}

	return aps, nil
}

// GetAllAPs returns all APs
func (ca *CacheAccessorImpl) GetAllAPs() ([]*APDevice, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var aps []*APDevice
	for _, orgData := range cache.Orgs {
		for _, ap := range orgData.Inventory.AP {
			apCopy := ap
			aps = append(aps, &apCopy)
		}
	}

	return aps, nil
}

// GetSwitchByMAC retrieves a switch by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetSwitchByMAC(mac string) (*MistSwitchDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	sw, exists := indexes.SwitchesByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("switch not found: %s", mac)
	}

	return sw, nil
}

// GetSwitchByName retrieves a switch by name with O(1) lookup
func (ca *CacheAccessorImpl) GetSwitchByName(name string) (*MistSwitchDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	sw, exists := indexes.SwitchesByName[name]
	if !exists {
		return nil, fmt.Errorf("switch not found: %s", name)
	}

	return sw, nil
}

// GetSwitchesBySite retrieves all switches for a specific site with O(1) lookup
func (ca *CacheAccessorImpl) GetSwitchesBySite(siteID string) ([]*MistSwitchDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	switches, exists := indexes.SwitchesBySite[siteID]
	if !exists {
		return []*MistSwitchDevice{}, nil
	}

	return switches, nil
}

// GetAllSwitches returns all switches
func (ca *CacheAccessorImpl) GetAllSwitches() ([]*MistSwitchDevice, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var switches []*MistSwitchDevice
	for _, orgData := range cache.Orgs {
		for _, sw := range orgData.Inventory.Switch {
			swCopy := sw
			switches = append(switches, &swCopy)
		}
	}

	return switches, nil
}

// GetGatewayByMAC retrieves a gateway by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetGatewayByMAC(mac string) (*MistGatewayDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	gw, exists := indexes.GatewaysByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("gateway not found: %s", mac)
	}

	return gw, nil
}

// GetGatewayByName retrieves a gateway by name with O(1) lookup
func (ca *CacheAccessorImpl) GetGatewayByName(name string) (*MistGatewayDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	gw, exists := indexes.GatewaysByName[name]
	if !exists {
		return nil, fmt.Errorf("gateway not found: %s", name)
	}

	return gw, nil
}

// GetGatewaysBySite retrieves all gateways for a specific site with O(1) lookup
func (ca *CacheAccessorImpl) GetGatewaysBySite(siteID string) ([]*MistGatewayDevice, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	gateways, exists := indexes.GatewaysBySite[siteID]
	if !exists {
		return []*MistGatewayDevice{}, nil
	}

	return gateways, nil
}

// GetAllGateways returns all gateways
func (ca *CacheAccessorImpl) GetAllGateways() ([]*MistGatewayDevice, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var gateways []*MistGatewayDevice
	for _, orgData := range cache.Orgs {
		for _, gw := range orgData.Inventory.Gateway {
			gwCopy := gw
			gateways = append(gateways, &gwCopy)
		}
	}

	return gateways, nil
}
