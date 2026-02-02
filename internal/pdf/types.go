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

// APConfig represents the configuration for a single Access Point
type APConfig struct {
	Name    string      // Full AP name (everything before first slash)
	Band24G *BandConfig // 2.4GHz band configuration
	Band5G  *BandConfig // 5GHz band configuration
	Band6G  *BandConfig // 6GHz band configuration
}

// BandConfig represents the configuration for a single radio band
type BandConfig struct {
	Channel string // Channel number or "auto" if 0
	Power   string // Power level or "auto" if empty
	Width   string // Channel width (default "20" if empty)
}

// Parser defines the interface for PDF parsing
type Parser interface {
	ParseFile(filePath string) ([]*APConfig, error)
}
