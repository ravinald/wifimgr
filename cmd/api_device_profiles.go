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
	"github.com/ravinald/wifimgr/internal/vendors"
)

// validateDeviceProfilesArgs validates arguments for the show api device-profiles command
func validateDeviceProfilesArgs(_ *cobra.Command, args []string) error {
	// Allow "help" as a special keyword
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return nil
		}
	}
	// Accept 0-2 arguments
	if len(args) > 2 {
		return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
	}
	return nil
}

// apiDeviceProfilesCmd represents the "show api device-profiles" command
var apiDeviceProfilesCmd = &cobra.Command{
	Use:   "device-profiles [profile-name] [no-resolve]",
	Short: "Show device profiles from API cache",
	Long: `Show device profiles retrieved from the local API cache.

Without arguments, displays all device profiles in a table format.
With a profile name argument, shows the raw JSON of the device profile details.

Arguments:
  profile-name - Optional profile name to show details for
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show api device-profiles              - Show all device profiles in table format
  wifimgr show api device-profiles AP-Profile   - Show specific profile details in JSON format
  wifimgr show api device-profiles no-resolve   - Show all profiles without field resolution`,
	Args: validateDeviceProfilesArgs,
	RunE: runShowAPIDeviceProfiles,
}

func init() {
	apiCmd.AddCommand(apiDeviceProfilesCmd)
}

func runShowAPIDeviceProfiles(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	logger := logging.GetLogger()
	logger.Info("Executing show api device-profiles command")

	// Get cache accessor
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		logger.WithError(err).Error("Failed to get cache accessor")
		return err
	}

	// Check for no-resolve in arguments
	noResolve := false
	profileName := ""

	for _, arg := range args {
		if arg == "no-resolve" {
			noResolve = true
		} else if profileName == "" {
			profileName = arg
		}
	}

	// If profile name is provided, show specific profile details
	if profileName != "" {
		return showProfileDetails(cacheAccessor, profileName)
	}

	// Otherwise, show all profiles in table format
	return showAllProfiles(cacheAccessor, noResolve)
}

func showProfileDetails(cacheAccessor *vendors.CacheAccessor, profileName string) error {
	logger := logging.GetLogger()

	// Get the device profile by name
	profile, err := cacheAccessor.GetDeviceProfileByName(profileName)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get device profile for %s", profileName)
		fmt.Printf("Profile not found: %s\n", profileName)
		return nil
	}

	// Marshal and print with color using MarshalJSONWithColorIndent
	jsonData, err := formatter.MarshalJSONWithColorIndent(profile, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to marshal profile to JSON")
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

func showAllProfiles(cacheAccessor *vendors.CacheAccessor, noResolve bool) error {
	// Get all device profiles
	profiles := cacheAccessor.GetAllDeviceProfiles()

	if len(profiles) == 0 {
		fmt.Println("No device profiles found in cache")
		return nil
	}

	// Sort profiles by name
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	// Create table data
	var tableData []formatter.GenericTableData

	for _, profile := range profiles {
		row := make(map[string]interface{})

		// Core fields (vendors.DeviceProfile uses plain strings)
		row["name"] = profile.Name
		row["type"] = strings.ToUpper(profile.Type)
		row["id"] = profile.ID

		// Site information
		if profile.ForSite {
			row["scope"] = "Site"
			if profile.SiteID != "" {
				// Try to resolve site name unless no-resolve is specified
				if !noResolve {
					if site, err := cacheAccessor.GetSiteByID(profile.SiteID); err == nil && site.Name != "" {
						row["site_org"] = site.Name
					} else {
						row["site_org"] = profile.SiteID
					}
				} else {
					row["site_org"] = profile.SiteID
				}
			} else {
				row["site_org"] = ""
			}
		} else {
			row["scope"] = "Org"
			// OrgID is stored but we don't have org name resolution in multi-vendor cache
			row["site_org"] = profile.OrgID
		}

		// Note: vendors.DeviceProfile doesn't have timestamps
		row["modified"] = ""
		row["created"] = ""

		tableData = append(tableData, formatter.GenericTableData(row))
	}

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name"},
		{Field: "type", Title: "Type"},
		{Field: "scope", Title: "Scope"},
		{Field: "site_org", Title: "Site/Org"},
		{Field: "id", Title: "Profile ID"},
		{Field: "created", Title: "Created"},
		{Field: "modified", Title: "Modified"},
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       fmt.Sprintf("Device Profiles (%d)", len(tableData)),
		Columns:     columns,
		Format:      "table",
		BoldHeaders: true,
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}
