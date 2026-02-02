/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package pdf

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"

	"github.com/ravinald/wifimgr/internal/logging"
)

// PDFParser implements the Parser interface using ledongthuc/pdf
type PDFParser struct {
	apRegex *regexp.Regexp
}

// NewParser creates a new PDF parser instance
func NewParser() *PDFParser {
	// Regex pattern from specification
	// Matches: @AP-NAME/band:channel:power:width (width optional)
	// AP configurations must be prefixed with @ to distinguish from drawing text
	// AP name is everything between @ and the first /
	// Values can be -1 (meaning auto) or undefined/empty
	pattern := `@([\w\-]+)((?:/(?:2:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40)?|5:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40|80|160)?|6:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40|80|160|320)?)){1,3})`

	return &PDFParser{
		apRegex: regexp.MustCompile(pattern),
	}
}

// ParseFile extracts AP configurations from a PDF file
func (p *PDFParser) ParseFile(filePath string) ([]*APConfig, error) {
	logging.Debugf("Opening PDF file: %s", filePath)

	// Open the PDF file
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logging.Warnf("Failed to close PDF file: %v", err)
		}
	}()

	// Get total pages
	totalPage := r.NumPage()
	logging.Debugf("PDF has %d pages", totalPage)

	// Extract text from all pages
	var allText strings.Builder
	for pageNum := 1; pageNum <= totalPage; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			logging.Warnf("Page %d is null, skipping", pageNum)
			continue
		}

		// Get text content from page
		content := page.Content()
		texts := content.Text

		// Build text, trying different approaches
		var textParts []string
		for _, text := range texts {
			if text.S != "" && text.S != " " {
				textParts = append(textParts, text.S)
			}
		}

		// Try without spaces (concatenated)
		pageText := strings.Join(textParts, "")
		if pageText != "" {
			allText.WriteString(pageText)
			allText.WriteString("\n")
		}

		// Try with spaces
		pageTextSpaced := strings.Join(textParts, " ")
		if pageTextSpaced != "" {
			allText.WriteString(pageTextSpaced)
			allText.WriteString("\n")
		}

		// Also try getting raw text directly
		if rawText, err := page.GetPlainText(nil); err == nil && rawText != "" {
			allText.WriteString(rawText)
			allText.WriteString("\n")
		}
	}

	fullText := allText.String()
	logging.Debugf("Extracted %d characters of text", len(fullText))

	// Find all AP configurations using regex
	matches := p.apRegex.FindAllStringSubmatch(fullText, -1)
	logging.Debugf("Found %d potential AP configurations", len(matches))

	// Parse matches into APConfig structs
	configMap := make(map[string]*APConfig)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		fullName := match[1] // AP name is everything before first slash
		radioSettings := match[2]

		// Create AP config
		config := &APConfig{
			Name: fullName,
		}

		// Parse radio settings
		p.parseRadioSettings(config, radioSettings)

		// Store in map to handle duplicates (keep first occurrence)
		if _, exists := configMap[fullName]; !exists {
			configMap[fullName] = config
			logging.Debugf("Added AP config: %s", fullName)
		}
	}

	// Convert map to slice for sorting
	configs := make([]*APConfig, 0, len(configMap))
	for _, config := range configMap {
		configs = append(configs, config)
	}

	// Natural sort the AP names
	sort.Slice(configs, func(i, j int) bool {
		return naturalCompare(configs[i].Name, configs[j].Name) < 0
	})

	logging.Infof("Successfully parsed %d unique AP configurations", len(configs))
	return configs, nil
}

// parseRadioSettings parses the radio settings string and populates band configurations
func (p *PDFParser) parseRadioSettings(config *APConfig, settings string) {

	// Remove leading slash if present
	settings = strings.TrimPrefix(settings, "/")

	// Split by forward slash to get individual band settings
	bands := strings.Split(settings, "/")

	for _, band := range bands {
		if band == "" || band == ":" {
			continue
		}

		// Parse band configuration
		parts := strings.Split(band, ":")
		if len(parts) < 1 {
			continue
		}

		bandType := parts[0]
		var channel, power, width string

		// Parse channel (index 1)
		if len(parts) > 1 && parts[1] != "" {
			if parts[1] == "0" || parts[1] == "-1" {
				channel = "auto"
			} else {
				channel = parts[1]
			}
		} else {
			channel = "auto"
		}

		// Parse power (index 2)
		if len(parts) > 2 && parts[2] != "" {
			if parts[2] == "-1" {
				power = "auto"
			} else {
				power = parts[2]
			}
		} else {
			power = "auto"
		}

		// Parse width (index 3)
		if len(parts) > 3 && parts[3] != "" {
			if parts[3] == "-1" {
				width = "" // Will be replaced with default later
			} else {
				width = parts[3]
			}
		} else {
			width = "" // Will be replaced with default later
		}

		bandConfig := &BandConfig{
			Channel: channel,
			Power:   power,
			Width:   width,
		}

		// Assign to appropriate band
		switch bandType {
		case "2":
			config.Band24G = bandConfig
		case "5":
			config.Band5G = bandConfig
		case "6":
			config.Band6G = bandConfig
		default:
			logging.Warnf("Unknown band type: %s", bandType)
		}
	}
}

// naturalCompare performs natural string comparison for sorting AP names
func naturalCompare(a, b string) int {
	// Split strings into parts (letters and numbers)
	partsA := splitNatural(a)
	partsB := splitNatural(b)

	minLen := len(partsA)
	if len(partsB) < minLen {
		minLen = len(partsB)
	}

	for i := 0; i < minLen; i++ {
		// Try to parse as numbers
		numA, errA := strconv.Atoi(partsA[i])
		numB, errB := strconv.Atoi(partsB[i])

		if errA == nil && errB == nil {
			// Both are numbers, compare numerically
			if numA != numB {
				return numA - numB
			}
		} else {
			// At least one is not a number, compare as strings
			if partsA[i] != partsB[i] {
				if partsA[i] < partsB[i] {
					return -1
				}
				return 1
			}
		}
	}

	// If all compared parts are equal, shorter string comes first
	return len(partsA) - len(partsB)
}

// splitNatural splits a string into alternating letter and number parts
func splitNatural(s string) []string {
	var parts []string
	var current strings.Builder
	var lastWasDigit bool

	for i, r := range s {
		isDigit := r >= '0' && r <= '9'

		if i > 0 && isDigit != lastWasDigit {
			// Transition between digit and non-digit
			parts = append(parts, current.String())
			current.Reset()
		}

		current.WriteRune(r)
		lastWasDigit = isDigit
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
