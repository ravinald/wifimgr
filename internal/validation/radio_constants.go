// Package validation provides configuration validation utilities.
package validation

// Band24Channels contains valid 2.4 GHz channels (US regulatory domain).
var Band24Channels = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}

// Band5Channels contains valid 5 GHz channels (UNII-1 through UNII-4).
var Band5Channels = []int{
	// UNII-1
	36, 40, 44, 48,
	// UNII-2A (DFS)
	52, 56, 60, 64,
	// UNII-2C (DFS)
	100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144,
	// UNII-3/4
	149, 153, 157, 161, 165, 169, 173, 177,
}

// Band6Channels contains valid 6 GHz channels (UNII-5 through UNII-8).
// Channels are spaced 4 apart (20MHz): 1, 5, 9, 13, ... up to 233.
var Band6Channels = generateBand6Channels()

func generateBand6Channels() []int {
	channels := make([]int, 0, 59)
	for ch := 1; ch <= 233; ch += 4 {
		channels = append(channels, ch)
	}
	return channels
}

// Band24Bandwidth contains valid bandwidths for 2.4 GHz (only 20MHz).
var Band24Bandwidth = []int{20}

// Band5Bandwidth contains valid bandwidths for 5 GHz (up to 160MHz).
var Band5Bandwidth = []int{20, 40, 80, 160}

// Band6Bandwidth contains valid bandwidths for 6 GHz (up to 320MHz for Wi-Fi 7).
var Band6Bandwidth = []int{20, 40, 80, 160, 320}

// PowerRange defines the valid transmit power range in dBm.
var PowerRange = struct {
	Min int
	Max int
}{1, 30}

// DualBandRadioModes defines valid radio_mode values by vendor.
var DualBandRadioModes = map[string][]int{
	"mist":   {24, 5},    // Mist: 2.4GHz or 5GHz (dual-band radios)
	"meraki": {5, 6},     // Meraki: 5GHz or 6GHz (flex radios)
	"":       {24, 5, 6}, // Unknown vendor: allow all
}

// channelSets provides quick lookup for valid channels by band.
var channelSets = struct {
	Band24 map[int]bool
	Band5  map[int]bool
	Band6  map[int]bool
}{}

// bandwidthSets provides quick lookup for valid bandwidths by band.
var bandwidthSets = struct {
	Band24 map[int]bool
	Band5  map[int]bool
	Band6  map[int]bool
}{}

func init() {
	// Initialize channel sets for O(1) lookup
	channelSets.Band24 = sliceToSet(Band24Channels)
	channelSets.Band5 = sliceToSet(Band5Channels)
	channelSets.Band6 = sliceToSet(Band6Channels)

	// Initialize bandwidth sets for O(1) lookup
	bandwidthSets.Band24 = sliceToSet(Band24Bandwidth)
	bandwidthSets.Band5 = sliceToSet(Band5Bandwidth)
	bandwidthSets.Band6 = sliceToSet(Band6Bandwidth)
}

func sliceToSet(slice []int) map[int]bool {
	set := make(map[int]bool, len(slice))
	for _, v := range slice {
		set[v] = true
	}
	return set
}

// IsValidChannel checks if a channel is valid for the given band.
func IsValidChannel(band string, channel int) bool {
	switch band {
	case "band_24", "24":
		return channelSets.Band24[channel]
	case "band_5", "5":
		return channelSets.Band5[channel]
	case "band_6", "6":
		return channelSets.Band6[channel]
	default:
		return false
	}
}

// IsValidBandwidth checks if a bandwidth is valid for the given band.
func IsValidBandwidth(band string, bandwidth int) bool {
	switch band {
	case "band_24", "24":
		return bandwidthSets.Band24[bandwidth]
	case "band_5", "5":
		return bandwidthSets.Band5[bandwidth]
	case "band_6", "6":
		return bandwidthSets.Band6[bandwidth]
	default:
		return false
	}
}

// IsValidPower checks if a power value is within the valid range.
func IsValidPower(power int) bool {
	return power >= PowerRange.Min && power <= PowerRange.Max
}

// IsValidRadioMode checks if a radio_mode is valid for the given vendor.
func IsValidRadioMode(vendor string, mode int) bool {
	allowedModes, ok := DualBandRadioModes[vendor]
	if !ok {
		allowedModes = DualBandRadioModes[""]
	}
	for _, m := range allowedModes {
		if m == mode {
			return true
		}
	}
	return false
}

// GetBandForRadioMode returns the band identifier (e.g., "band_24") for a radio_mode value.
func GetBandForRadioMode(mode int) string {
	switch mode {
	case 24:
		return "band_24"
	case 5:
		return "band_5"
	case 6:
		return "band_6"
	default:
		return ""
	}
}

// GetValidChannels returns the list of valid channels for a band.
func GetValidChannels(band string) []int {
	switch band {
	case "band_24", "24":
		return Band24Channels
	case "band_5", "5":
		return Band5Channels
	case "band_6", "6":
		return Band6Channels
	default:
		return nil
	}
}

// GetValidBandwidths returns the list of valid bandwidths for a band.
func GetValidBandwidths(band string) []int {
	switch band {
	case "band_24", "24":
		return Band24Bandwidth
	case "band_5", "5":
		return Band5Bandwidth
	case "band_6", "6":
		return Band6Bandwidth
	default:
		return nil
	}
}
