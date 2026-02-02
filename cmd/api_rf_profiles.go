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
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// apiRFProfilesCmd represents the "show api rf-profiles" command
var apiRFProfilesCmd = &cobra.Command{
	Use:   "rf-profiles [profile-name] [site <site-name>] [json|csv] [no-resolve]",
	Short: "Show RF profiles from API cache",
	Long: `Show RF profiles retrieved from the local API cache.

Without arguments, displays all RF profiles in a table format.
With a profile name argument, shows the raw JSON of the RF profile details.

Arguments:
  profile-name - Optional profile name or ID to filter by
  site         - Filter by site name (use "site <site-name>")
  json         - Output in JSON format
  csv          - Output in CSV format
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show api rf-profiles                       - Show all RF profiles in table format
  wifimgr show api rf-profiles "Basic Indoor"        - Show specific profile details in JSON
  wifimgr show api rf-profiles site US-LAB-01        - Show RF profiles for a specific site
  wifimgr show api rf-profiles json                  - Show all profiles in JSON format
  wifimgr show api rf-profiles csv                   - Show all profiles in CSV format
  wifimgr show api rf-profiles no-resolve            - Show all profiles without field resolution`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return nil
	},
	RunE: runShowAPIRFProfiles,
}

func init() {
	apiCmd.AddCommand(apiRFProfilesCmd)
}

func runShowAPIRFProfiles(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	logger := logging.GetLogger()
	logger.Info("Executing show api rf-profiles command")

	// Get cache accessor
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		logger.WithError(err).Error("Failed to get cache accessor")
		return err
	}

	// Parse arguments
	var profileFilter string
	var siteFilter string
	format := "table"
	noResolve := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "json":
			format = "json"
		case "csv":
			format = "csv"
		case "no-resolve":
			noResolve = true
		case "site":
			if i+1 < len(args) {
				i++
				siteFilter = args[i]
			}
		default:
			if profileFilter == "" {
				profileFilter = arg
			}
		}
	}

	// Get all RF profiles
	profiles := cacheAccessor.GetAllRFTemplates()

	if len(profiles) == 0 {
		fmt.Println("No RF profiles found in cache")
		return nil
	}

	// Filter by site if specified
	if siteFilter != "" {
		var filtered []*vendors.RFTemplate
		for _, p := range profiles {
			// Try to match site by name or ID
			if p.SiteID != "" {
				if patterns.Contains(p.SiteID, siteFilter) {
					filtered = append(filtered, p)
				} else if !noResolve {
					// Try to resolve site name
					if site, err := cacheAccessor.GetSiteByID(p.SiteID); err == nil && site.Name != "" {
						if patterns.Contains(site.Name, siteFilter) {
							filtered = append(filtered, p)
						}
					}
				}
			}
		}
		profiles = filtered
	}

	// Filter by profile name/ID if specified
	if profileFilter != "" {
		var filtered []*vendors.RFTemplate
		for _, p := range profiles {
			if patterns.Contains(p.Name, profileFilter) || patterns.Contains(p.ID, profileFilter) {
				filtered = append(filtered, p)
			}
		}
		profiles = filtered

		// If single profile matched and not csv format, show JSON details
		if len(profiles) == 1 && format != "csv" {
			return showRFProfileDetails(profiles[0])
		}
	}

	if len(profiles) == 0 {
		if profileFilter != "" {
			fmt.Printf("No RF profiles found matching '%s'\n", profileFilter)
		} else if siteFilter != "" {
			fmt.Printf("No RF profiles found for site '%s'\n", siteFilter)
		}
		return nil
	}

	// Sort profiles by name
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	// Handle JSON output
	if format == "json" {
		return outputRFProfilesJSON(profiles)
	}

	// Build table data
	return outputRFProfilesTable(profiles, cacheAccessor, noResolve, format)
}

func showRFProfileDetails(profile *vendors.RFTemplate) error {
	// Marshal and print with color using MarshalJSONWithColorIndent
	jsonData, err := formatter.MarshalJSONWithColorIndent(profile, "", "  ")
	if err != nil {
		logging.GetLogger().WithError(err).Error("Failed to marshal RF profile to JSON")
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputRFProfilesJSON(profiles []*vendors.RFTemplate) error {
	var jsonData []byte
	var err error

	if len(profiles) == 1 {
		jsonData, err = formatter.MarshalJSONWithColorIndent(profiles[0], "", "  ")
	} else {
		jsonData, err = formatter.MarshalJSONWithColorIndent(profiles, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputRFProfilesTable(profiles []*vendors.RFTemplate, cacheAccessor *vendors.CacheAccessor, noResolve bool, format string) error {
	var tableData []formatter.GenericTableData

	for _, profile := range profiles {
		row := make(map[string]interface{})

		row["name"] = profile.Name
		row["id"] = profile.ID

		// Site information
		if profile.SiteID != "" {
			if !noResolve {
				if site, err := cacheAccessor.GetSiteByID(profile.SiteID); err == nil && site.Name != "" {
					row["site"] = site.Name
				} else {
					row["site"] = profile.SiteID
				}
			} else {
				row["site"] = profile.SiteID
			}
		} else {
			row["site"] = ""
		}

		// Extract key settings from config
		if profile.Config != nil {
			// Band selection
			if bandType, ok := profile.Config["bandSelectionType"].(string); ok {
				row["band_selection"] = bandType
			} else {
				row["band_selection"] = ""
			}

			// Client balancing
			if clientBal, ok := profile.Config["clientBalancingEnabled"].(bool); ok {
				if clientBal {
					row["client_balancing"] = "Yes"
				} else {
					row["client_balancing"] = "No"
				}
			} else {
				row["client_balancing"] = ""
			}

			// Min bitrate type
			if minBitrate, ok := profile.Config["minBitrateType"].(string); ok {
				row["min_bitrate_type"] = minBitrate
			} else {
				row["min_bitrate_type"] = ""
			}

			// 2.4GHz settings summary
			if twoFour, ok := profile.Config["twoFourGhzSettings"].(map[string]interface{}); ok {
				var settings []string
				if minPower, ok := twoFour["minPower"].(float64); ok {
					settings = append(settings, fmt.Sprintf("min:%ddBm", int(minPower)))
				}
				if maxPower, ok := twoFour["maxPower"].(float64); ok {
					settings = append(settings, fmt.Sprintf("max:%ddBm", int(maxPower)))
				}
				if len(settings) > 0 {
					row["2.4ghz"] = fmt.Sprintf("%v", settings)
				} else {
					row["2.4ghz"] = ""
				}
			} else {
				row["2.4ghz"] = ""
			}

			// 5GHz settings summary
			if fiveGhz, ok := profile.Config["fiveGhzSettings"].(map[string]interface{}); ok {
				var settings []string
				if minPower, ok := fiveGhz["minPower"].(float64); ok {
					settings = append(settings, fmt.Sprintf("min:%ddBm", int(minPower)))
				}
				if maxPower, ok := fiveGhz["maxPower"].(float64); ok {
					settings = append(settings, fmt.Sprintf("max:%ddBm", int(maxPower)))
				}
				if channelWidth, ok := fiveGhz["channelWidth"].(string); ok {
					settings = append(settings, channelWidth)
				}
				if len(settings) > 0 {
					row["5ghz"] = fmt.Sprintf("%v", settings)
				} else {
					row["5ghz"] = ""
				}
			} else {
				row["5ghz"] = ""
			}
		} else {
			row["band_selection"] = ""
			row["client_balancing"] = ""
			row["min_bitrate_type"] = ""
			row["2.4ghz"] = ""
			row["5ghz"] = ""
		}

		tableData = append(tableData, formatter.GenericTableData(row))
	}

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name"},
		{Field: "site", Title: "Site"},
		{Field: "band_selection", Title: "Band Selection"},
		{Field: "client_balancing", Title: "Client Bal."},
		{Field: "2.4ghz", Title: "2.4GHz"},
		{Field: "5ghz", Title: "5GHz"},
		{Field: "id", Title: "Profile ID"},
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       fmt.Sprintf("RF Profiles (%d)", len(tableData)),
		Columns:     columns,
		Format:      format,
		BoldHeaders: true,
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}
