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

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/fieldinfo"
	"github.com/ravinald/wifimgr/internal/formatter"
)

// showFieldsCmd represents the "show fields" command
var showFieldsCmd = &cobra.Command{
	Use:   "fields [command-path]",
	Short: "List available display fields for a command",
	Long: `List all available fields that can be configured in the display
section of wifimgr-config.json for the specified command.

Examples:
  wifimgr show fields show.api.ap
  wifimgr show fields show.inventory.switch
  wifimgr show fields show.api.sites

To see all commands with configurable fields:
  wifimgr show fields`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShowFields,
}

func runShowFields(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return listAllCommands()
	}

	cmdPath := args[0]
	fields, err := fieldinfo.GetFieldsForCommand(cmdPath)
	if err != nil {
		return err
	}

	// Sort fields alphabetically
	sort.Slice(fields.Fields, func(i, j int) bool {
		return fields.Fields[i].Name < fields.Fields[j].Name
	})

	// Print header
	fmt.Printf("\nAvailable fields for '%s' (type: %s):\n\n", fields.CommandPath, fields.DataType)

	// Build table data
	var tableData []formatter.GenericTableData
	for _, f := range fields.Fields {
		tableData = append(tableData, formatter.GenericTableData{
			"field": f.Name,
			"type":  f.Type,
		})
	}

	// Render as table
	config := formatter.TableConfig{
		Format:      "table",
		Title:       "",
		BoldHeaders: true,
		Columns: []formatter.TableColumn{
			{Field: "field", Title: "Field Name", MaxWidth: 25},
			{Field: "type", Title: "Type", MaxWidth: 15},
		},
	}

	printer := formatter.NewGenericTablePrinter(config, tableData)
	fmt.Print(printer.Print())

	// Print config hint
	fmt.Printf("\nConfigure in wifimgr-config.json:\n")
	fmt.Printf("  display.commands[\"%s\"].fields\n\n", cmdPath)

	return nil
}

func listAllCommands() error {
	commands := fieldinfo.ListCommands()

	fmt.Println("\nCommands with configurable display fields:")
	fmt.Println()
	for _, cmd := range commands {
		fmt.Printf("  %s\n", cmd)
	}
	fmt.Println("\nUse 'wifimgr show fields <command>' to see available fields.")
	return nil
}

func init() {
	showCmd.AddCommand(showFieldsCmd)
}
