package formatter

import (
	"bytes"

	"github.com/neilotoole/jsoncolor"
	"github.com/spf13/viper"
)

// GetJSONColorConfig returns the jsoncolor.Colors configuration from Viper
func GetJSONColorConfig() *jsoncolor.Colors {
	// Start with default colors
	colors := jsoncolor.DefaultColors()

	// Override with colors from config if they are set
	if hexColor := viper.GetString("display.jsoncolor.null.hex"); hexColor != "" {
		colors.Null = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.bool.hex"); hexColor != "" {
		colors.Bool = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.number.hex"); hexColor != "" {
		colors.Number = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.string.hex"); hexColor != "" {
		colors.String = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.key.hex"); hexColor != "" {
		colors.Key = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.bytes.hex"); hexColor != "" {
		colors.Bytes = hexToAnsiColor(hexColor)
	}
	if hexColor := viper.GetString("display.jsoncolor.time.hex"); hexColor != "" {
		colors.Time = hexToAnsiColor(hexColor)
	}

	return colors
}

// hexToAnsiColor converts a hex color to an ANSI color escape sequence
// This is a simplified implementation - for full color support you might want
// to use a more sophisticated color conversion library
func hexToAnsiColor(hex string) jsoncolor.Color {
	// For now, map common hex colors to ANSI colors
	switch hex {
	case "#767676":
		return jsoncolor.Color("\x1b[90m") // Bright black (gray)
	case "#FFFFFF":
		return jsoncolor.Color("\x1b[97m") // Bright white
	case "#00FFFF":
		return jsoncolor.Color("\x1b[96m") // Bright cyan
	case "#00FF00":
		return jsoncolor.Color("\x1b[92m") // Bright green
	case "#0000FF":
		return jsoncolor.Color("\x1b[94m") // Bright blue
	default:
		// Default to no color for unknown hex values
		return jsoncolor.Color{}
	}
}

// MarshalJSONWithColorIndent marshals data with colors and indentation
func MarshalJSONWithColorIndent(data interface{}, prefix, indent string) ([]byte, error) {
	// Create a buffer to capture the output
	var buf bytes.Buffer

	// Create encoder with our colors
	enc := jsoncolor.NewEncoder(&buf)
	enc.SetColors(GetJSONColorConfig())
	enc.SetIndent(prefix, indent)

	// Encode the data
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}

	// Return the buffer contents, removing the trailing newline added by Encode
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}
