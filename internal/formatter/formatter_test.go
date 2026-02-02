package formatter

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/ravinald/wifimgr/internal/config"
)

func TestFormatAsTableWithColoredHeaders(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name        string `json:"name"`
		ID          string `json:"id"`
		Description string `json:"description"`
	}

	testData := []interface{}{
		TestStruct{
			Name:        "Test1",
			ID:          "123456",
			Description: "This is a test item with a fairly long description",
		},
		TestStruct{
			Name:        "TestItem2",
			ID:          "7890",
			Description: "Short desc",
		},
	}

	// Define display format
	displayFormat := config.DisplayFormat{
		Format: "table",
		Fields: []string{"name", "id", "description"},
	}

	// Temporarily disable colors for deterministic testing
	origNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = origNoColor }()

	// Format as table
	result := formatAsTable(testData, displayFormat)

	// Verify that we have the correct number of lines
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 4 { // header + separator + 2 data rows
		t.Errorf("Expected 4 lines in the output, got %d", len(lines))
	}

	// Get all lines for testing
	// headerLine contains ANSI color codes so we'll use it indirectly
	separatorLine := lines[1]
	dataLine1 := lines[2]
	dataLine2 := lines[3]

	// Verify column alignment - find locations of second column starts

	// The separator line will have "  -" at the start of the second column
	secondColSepStart := strings.Index(separatorLine, "  -")
	if secondColSepStart <= 0 {
		t.Errorf("Couldn't find start of second column in separator line")
		return
	}

	// Check data line alignment
	// Since the colored header line contains ANSI codes, we check the substring starting at the
	// position where the second column should be
	if !strings.Contains(dataLine1[secondColSepStart:secondColSepStart+10], "123456") {
		t.Errorf("First data row second column is misaligned. Second column data should start around position %d", secondColSepStart)
	}

	if !strings.Contains(dataLine2[secondColSepStart:secondColSepStart+10], "7890") {
		t.Errorf("Second data row second column is misaligned. Second column data should start around position %d", secondColSepStart)
	}

	// Find third column starting position
	thirdColSepStart := strings.Index(separatorLine[secondColSepStart+1:], "  -")
	if thirdColSepStart <= 0 {
		t.Errorf("Couldn't find start of third column in separator line")
		return
	}
	thirdColSepStart += secondColSepStart + 1 // Adjust for the substring search

	// Check third column alignment
	if !strings.Contains(dataLine1[thirdColSepStart:], "This is a test") {
		t.Errorf("First data row third column is misaligned. Should start around position %d", thirdColSepStart)
	}

	if !strings.Contains(dataLine2[thirdColSepStart:], "Short desc") {
		t.Errorf("Second data row third column is misaligned. Should start around position %d", thirdColSepStart)
	}
}

func TestFormatAsCSV(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name        string `json:"name"`
		ID          string `json:"id"`
		Description string `json:"description"`
	}

	testData := []interface{}{
		TestStruct{
			Name:        "Test1",
			ID:          "123456",
			Description: "This is a test item",
		},
		TestStruct{
			Name:        "Test2",
			ID:          "7890",
			Description: "Item with, comma",
		},
	}

	// Define display format
	displayFormat := config.DisplayFormat{
		Format: "csv",
		Fields: []string{"name", "id", "description"},
	}

	// Temporarily disable colors for deterministic testing
	origNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = origNoColor }()

	// Format as CSV
	result := formatAsCSV(testData, displayFormat)

	// Verify results
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 lines in CSV output, got %d", len(lines))
	}

	// Verify header
	if !strings.Contains(lines[0], "name,id,description") {
		t.Errorf("Header row is incorrect: %s", lines[0])
	}

	// Verify first data row
	expected1 := "Test1,123456,This is a test item"
	if lines[1] != expected1 {
		t.Errorf("First data row is incorrect, got: %s, expected: %s", lines[1], expected1)
	}

	// Verify second data row with comma
	expected2 := `Test2,7890,"Item with, comma"`
	if lines[2] != expected2 {
		t.Errorf("Second data row is incorrect, got: %s, expected: %s", lines[2], expected2)
	}
}
