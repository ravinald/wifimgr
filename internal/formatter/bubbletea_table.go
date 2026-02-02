package formatter

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/symbols"
)

// BubbleTableModel wraps the bubbles table.Model with our configuration
type BubbleTableModel struct {
	table        table.Model
	config       TableConfig
	data         []GenericTableData
	interactive  bool
	quit         bool
	csvSupported bool
	ready        bool
	windowWidth  int
	windowHeight int
}

// NewBubbleTable creates a new BubbleTea table with the given configuration
func NewBubbleTable(config TableConfig, data []GenericTableData, interactive bool) *BubbleTableModel {
	// Convert our column definitions to bubbles table columns
	var columns []table.Column
	visibleColumns := []TableColumn{}

	for _, col := range config.Columns {
		if !col.IsHidden {
			visibleColumns = append(visibleColumns, col)

			// Get display title
			title := col.Title
			if title == "" {
				title = col.Header
			}

			// Calculate width based on MaxWidth configuration:
			// width = -1: Auto-size to content, no scaling
			// width = 0: Auto-size to content, then scale to fit terminal
			// width > 0: Use exact specified width
			width := col.MaxWidth
			if width <= 0 {
				// For auto-sizing columns (width <= 0), don't calculate fixed width here
				// Let the static renderer handle dynamic width allocation
				width = 1 // Minimal placeholder width for BubbleTea table
			}

			columns = append(columns, table.Column{
				Title: title,
				Width: width,
			})
		}
	}

	// Convert data to table rows
	var rows []table.Row
	for _, item := range data {
		row := make(table.Row, len(visibleColumns))
		for i, col := range visibleColumns {
			var val interface{}
			var ok bool

			// Check for cache.* field path (e.g., "cache.radio_config.band_5.channel")
			if strings.HasPrefix(col.Field, "cache.") && config.CacheAccess != nil {
				// Get MAC address from item to look up cache data
				if mac, hasMac := item["mac"].(string); hasMac && mac != "" {
					if cachedData, found := config.CacheAccess.GetCachedData(mac); found {
						// Extract the path after "cache." prefix
						cachePath := strings.TrimPrefix(col.Field, "cache.")
						val, ok = config.CacheAccess.GetFieldByPath(cachedData, cachePath)
					}
				}
			} else {
				// Direct field access
				val, ok = item[col.Field]
			}

			if !ok {
				row[i] = ""
				continue
			}

			// Field resolution now done during data preparation

			// Format value
			var strVal string
			if col.IsBoolField {
				if bVal, ok := val.(bool); ok {
					// Simple boolean logic: true→"C"/"Yes", false→"D"/"No"
					if col.IsConnectionField {
						if bVal {
							strVal = "CONN_TRUE"
						} else {
							strVal = "CONN_FALSE"
						}
					} else {
						if bVal {
							strVal = "BOOL_TRUE"
						} else {
							strVal = "BOOL_FALSE"
						}
					}
				} else {
					// Handle nil/undefined values: nil→"?" for both connection and regular boolean fields
					if col.IsConnectionField {
						strVal = "CONN_UNKNOWN"
					} else {
						strVal = "BOOL_UNKNOWN"
					}
				}
			} else if strings.HasPrefix(col.Field, "cache.") {
				// Use formatNestedValue for cache.* fields to handle complex nested values
				strVal = formatNestedValue(val)
			} else {
				strVal = fmt.Sprintf("%v", val)
			}

			// Apply truncation if needed
			if col.MaxWidth > 0 && lipgloss.Width(strVal) > col.MaxWidth {
				// For styled strings, we need to truncate based on display width, not string length
				if col.MaxWidth > 3 {
					truncated := ""
					currentWidth := 0
					targetWidth := col.MaxWidth - 3 // Leave space for "..."

					// Convert string to runes to handle multi-byte characters properly
					for _, r := range strVal {
						charStr := string(r)
						charWidth := lipgloss.Width(charStr)
						if currentWidth+charWidth > targetWidth {
							break
						}
						truncated += charStr
						currentWidth += charWidth
					}
					strVal = truncated + "..."
				} else if col.MaxWidth > 0 {
					// For very small widths, just truncate to the width without ellipsis
					truncated := ""
					currentWidth := 0
					for _, r := range strVal {
						charStr := string(r)
						charWidth := lipgloss.Width(charStr)
						if currentWidth+charWidth > col.MaxWidth {
							break
						}
						truncated += charStr
						currentWidth += charWidth
					}
					strVal = truncated
				}
			}

			row[i] = strVal
		}
		rows = append(rows, row)
	}

	// Create the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(interactive),
		table.WithHeight(10), // Initial height, will be adjusted
	)

	// Apply styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(config.BoldHeaders)

	if interactive {
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
	}

	t.SetStyles(s)

	return &BubbleTableModel{
		table:        t,
		config:       config,
		data:         data,
		interactive:  interactive,
		csvSupported: true,
	}
}

// Init initializes the model
func (m BubbleTableModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m BubbleTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		// Calculate available height for table
		titleHeight := 0
		if m.config.Title != "" {
			titleHeight = 2 // Title + blank line
		}
		footerHeight := 3 // Blank line + help text + blank line

		// The table component handles its own header, so we just need to account for
		// title and footer space
		availableHeight := msg.Height - titleHeight - footerHeight

		// Set table dimensions
		// The table will show the header plus this many data rows
		m.table.SetHeight(availableHeight - 1) // -1 for the header row
		m.table.SetWidth(msg.Width)
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		}
	}

	// Update table
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the table
func (m BubbleTableModel) View() string {
	if m.quit {
		return ""
	}

	if !m.ready && m.interactive {
		return "\n  Initializing..."
	}

	var output strings.Builder

	// Add title if provided
	if m.config.Title != "" {
		output.WriteString(m.config.Title)
		output.WriteString("\n\n")
	}

	// Render the table (includes header and data with built-in scrolling)
	output.WriteString(m.table.View())

	// Add help text for interactive mode
	if m.interactive {
		output.WriteString("\n\n  Up/Down: Navigate | Enter: Select | q: Quit")
	}

	return output.String()
}

// RenderStatic renders the table without interactivity
func (m *BubbleTableModel) RenderStatic() string {
	// Get terminal width directly using golang.org/x/term
	termWidth := 80 // default fallback

	// Check if file descriptors are actual terminals
	stdoutFd := int(os.Stdout.Fd())
	stdinFd := int(os.Stdin.Fd())
	stderrFd := int(os.Stderr.Fd())

	// Try stdout first (if it's a terminal)
	if term.IsTerminal(stdoutFd) {
		if width, _, err := term.GetSize(stdoutFd); err == nil && width > 0 {
			termWidth = width
		}
	} else {
		// Fallback to stdin (if it's a terminal)
		if term.IsTerminal(stdinFd) {
			if width, _, err := term.GetSize(stdinFd); err == nil && width > 0 {
				termWidth = width
			}
		} else {
			// Fallback to stderr (if it's a terminal)
			if term.IsTerminal(stderrFd) {
				if width, _, err := term.GetSize(stderrFd); err == nil && width > 0 {
					termWidth = width
				}
			} else {
				// Final fallback to environment variables or reasonable defaults
				if cols := os.Getenv("COLUMNS"); cols != "" {
					if envWidth, err := strconv.Atoi(cols); err == nil && envWidth > 0 {
						termWidth = envWidth
					}
				} else {
					// When no terminal is available, use a more generous default
					// This handles cases where output is redirected but we still want reasonable formatting
					termWidth = 120 // More reasonable default for modern terminals
				}
			}
		}
	}

	// For static rendering, we'll use a simplified view
	var output strings.Builder

	// Add title if provided
	if m.config.Title != "" {
		output.WriteString(m.config.Title)
		output.WriteString("\n\n")
	}

	// Get visible columns
	var visibleColumns []TableColumn
	for _, col := range m.config.Columns {
		if !col.IsHidden {
			visibleColumns = append(visibleColumns, col)
		}
	}

	// Calculate column widths
	colWidths := make([]int, len(visibleColumns))
	for i, col := range visibleColumns {
		title := col.Title
		if title == "" {
			title = col.Header
		}
		colWidths[i] = lipgloss.Width(title)

		// Check data width
		for _, item := range m.data {
			// Direct field access
			val, ok := item[col.Field]

			if ok {
				strVal := fmt.Sprintf("%v", val)
				if col.IsBoolField {
					if bVal, ok := val.(bool); ok {
						strVal = symbols.FormatBooleanValue(bVal, col.IsConnectionField)
					}
				} else if col.IsStatusField {
					// Convert status to marker for width calculation
					strVal = fmt.Sprintf("STATUS_%s", strings.ToUpper(strVal))
				}

				// For width calculation, get the plain text content that will be displayed
				displayContent := strVal
				switch strVal {
				case "CONN_TRUE":
					displayContent = "C"
				case "CONN_FALSE":
					displayContent = "D"
				case "CONN_UNKNOWN":
					displayContent = "?"
				case "BOOL_TRUE":
					displayContent = "Yes"
				case "BOOL_FALSE":
					displayContent = "No"
				case "STATUS_ONLINE":
					displayContent = "C"
				case "STATUS_OFFLINE":
					displayContent = "D"
				case "STATUS_ALERTING":
					displayContent = "A"
				case "STATUS_DORMANT":
					displayContent = "Z"
				default:
					if strings.HasPrefix(strVal, "GREEN_TEXT:") {
						displayContent = strings.TrimPrefix(strVal, "GREEN_TEXT:")
					} else if strings.HasPrefix(strVal, "STATUS_") {
						displayContent = strings.TrimPrefix(strVal, "STATUS_")
					}
				}

				// Use lipgloss.Width on the plain content (without ANSI codes)
				if lipgloss.Width(displayContent) > colWidths[i] {
					colWidths[i] = lipgloss.Width(displayContent)
				}
			}
		}

		// Apply MaxWidth constraint if specified (only for positive values)
		if col.MaxWidth > 0 && colWidths[i] > col.MaxWidth {
			colWidths[i] = col.MaxWidth
		}

	}

	// Adjust column widths to fit terminal width
	totalWidth := 0
	for i := range colWidths {
		totalWidth += colWidths[i] + 2 // +2 for padding
	}

	// Scale columns to use available terminal width based on width configuration:
	// width = -1: Fit largest content, no scaling (auto-size)
	// width = 0: Fit largest content, then scale to fit terminal
	// width > 0: Use exact specified width, no scaling
	if totalWidth != termWidth && termWidth > 20 { // Only scale if we have reasonable terminal width
		// Calculate scale factor
		availableWidth := termWidth - len(colWidths)*2 // Account for padding
		currentContentWidth := totalWidth - len(colWidths)*2

		if currentContentWidth > 0 {
			scaleFactor := float64(availableWidth) / float64(currentContentWidth)
			// Apply scaling based on width configuration
			for i, col := range visibleColumns {
				// width = -1: Auto-size to content, no scaling
				if col.MaxWidth == -1 {
					// Keep the calculated width, don't scale
					continue
				}

				// width > 0: Fixed width, no scaling
				if col.MaxWidth > 0 {
					// Keep the configured width, don't scale
					continue
				}

				// width = 0: Auto-size to content, then scale to fit terminal
				newWidth := int(float64(colWidths[i]) * scaleFactor)

				// Keep minimum width of 5 for readability
				if newWidth < 5 {
					newWidth = 5
				}

				// For very large scale factors, cap the width to prevent excessive spacing
				maxWidth := colWidths[i] * 3 // Don't expand more than 3x original
				if newWidth > maxWidth && scaleFactor > 1 {
					newWidth = maxWidth
				}

				colWidths[i] = newWidth
			}
		}
	}

	// Print headers
	for i, col := range visibleColumns {
		title := col.Title
		if title == "" {
			title = col.Header
		}

		format := fmt.Sprintf("%%-%ds", colWidths[i]+2)
		header := fmt.Sprintf(format, title)

		if m.config.BoldHeaders {
			header = lipgloss.NewStyle().Bold(true).Render(title)
			// Ensure padding count is never negative
			paddingCount := colWidths[i] + 2 - lipgloss.Width(title)
			if paddingCount < 0 {
				paddingCount = 0
			}
			padding := strings.Repeat(" ", paddingCount)
			output.WriteString(header + padding)
		} else {
			output.WriteString(header)
		}
	}
	output.WriteString("\n")

	// Print separator if enabled
	if m.config.ShowSeparator {
		for i, width := range colWidths {
			// Ensure width is never negative
			if width < 0 {
				width = 0
			}
			output.WriteString(strings.Repeat("-", width))
			if i < len(colWidths)-1 {
				output.WriteString("  ")
			}
		}
		output.WriteString("\n")
	}

	// Define alternating row background color using adaptive colors
	altRowBgColor := lipgloss.AdaptiveColor{
		Light: "#f8f9fa", // Light gray for dark text on light backgrounds
		Dark:  "#1a1a1a", // Dark gray for light text on dark backgrounds
	}

	// Print data rows
	for rowIdx, item := range m.data {
		isAlternateRow := rowIdx%2 == 1

		for i, col := range visibleColumns {
			var val interface{}
			var exists bool

			// Check for cache.* field path (e.g., "cache.radio_config.band_5.channel")
			if strings.HasPrefix(col.Field, "cache.") && m.config.CacheAccess != nil {
				// Get MAC address from item to look up cache data
				if mac, hasMac := item["mac"].(string); hasMac && mac != "" {
					if cachedData, found := m.config.CacheAccess.GetCachedData(mac); found {
						// Extract the path after "cache." prefix
						cachePath := strings.TrimPrefix(col.Field, "cache.")
						val, exists = m.config.CacheAccess.GetFieldByPath(cachedData, cachePath)
					}
				}
			} else {
				// Direct field access
				val, exists = item[col.Field]
			}

			if !exists {
				// For empty fields, just add the appropriate spacing
				cellContent := strings.Repeat(" ", colWidths[i]+2)
				if isAlternateRow {
					cellContent = lipgloss.NewStyle().Background(altRowBgColor).Render(cellContent)
				}
				output.WriteString(cellContent)
				continue
			}

			// Format value
			var strVal string
			if col.IsBoolField {
				if bVal, ok := val.(bool); ok {
					// Simple boolean logic: true→"C"/"Yes", false→"D"/"No"
					if col.IsConnectionField {
						if bVal {
							strVal = "CONN_TRUE"
						} else {
							strVal = "CONN_FALSE"
						}
					} else {
						if bVal {
							strVal = "BOOL_TRUE"
						} else {
							strVal = "BOOL_FALSE"
						}
					}
				} else {
					// Handle nil/undefined values: nil→"?" for both connection and regular boolean fields
					if col.IsConnectionField {
						strVal = "CONN_UNKNOWN"
					} else {
						strVal = "BOOL_UNKNOWN"
					}
				}
			} else if col.IsStatusField {
				// Handle status field values: online/offline/alerting/dormant
				strVal = fmt.Sprintf("STATUS_%s", strings.ToUpper(fmt.Sprintf("%v", val)))
			} else if strings.HasPrefix(col.Field, "cache.") {
				// Use formatNestedValue for cache.* fields to handle complex nested values
				strVal = formatNestedValue(val)
			} else {
				strVal = fmt.Sprintf("%v", val)
			}

			// Evaluate symbols at render time for proper terminal detection
			// Store whether this cell has foreground color for background application
			var hasColor bool
			var originalContent string // Track the original content for truncation
			var styleType string       // Track what type of styling to apply

			switch strVal {
			case "CONN_TRUE":
				originalContent = "C"
				styleType = "green"
				hasColor = true
			case "CONN_FALSE":
				originalContent = "D"
				styleType = "red"
				hasColor = true
			case "CONN_UNKNOWN":
				originalContent = "?"
				styleType = "blue"
				hasColor = true
			case "BOOL_TRUE":
				originalContent = "Yes"
				styleType = "none"
			case "BOOL_FALSE":
				originalContent = "No"
				styleType = "none"
			case "BOOL_UNKNOWN":
				originalContent = "?"
				styleType = "blue"
				hasColor = true
			case "STATUS_ONLINE":
				originalContent = "C" // Connected symbol
				styleType = "green"
				hasColor = true
			case "STATUS_OFFLINE":
				originalContent = "D" // Disconnected symbol
				styleType = "none"    // white/default
			case "STATUS_ALERTING":
				originalContent = "A" // Alerting symbol
				styleType = "yellow"
				hasColor = true
			case "STATUS_DORMANT":
				originalContent = "Z" // Dormant symbol (Z for sleep/zzz)
				styleType = "blue"
				hasColor = true
			default:
				// Check for GREEN_TEXT: prefix
				if strings.HasPrefix(strVal, "GREEN_TEXT:") {
					originalContent = strings.TrimPrefix(strVal, "GREEN_TEXT:")
					styleType = "green"
					hasColor = true
				} else if strings.HasPrefix(strVal, "STATUS_") {
					// Unknown status - show as-is
					originalContent = strings.TrimPrefix(strVal, "STATUS_")
					styleType = "none"
				} else {
					originalContent = strVal
					styleType = "none"
				}
			}

			// Apply truncation based on calculated column width using the original content
			if lipgloss.Width(originalContent) > colWidths[i] {
				if colWidths[i] > 3 {
					// Truncate the original content first
					truncated := ""
					currentWidth := 0
					targetWidth := colWidths[i] - 3 // Leave space for "..."

					// Convert string to runes to handle multi-byte characters properly
					for _, r := range originalContent {
						charStr := string(r)
						charWidth := lipgloss.Width(charStr)
						if currentWidth+charWidth > targetWidth {
							break
						}
						truncated += charStr
						currentWidth += charWidth
					}
					originalContent = truncated + "..."
				} else if colWidths[i] > 0 {
					// For very small widths, just truncate to the width without ellipsis
					truncated := ""
					currentWidth := 0
					for _, r := range originalContent {
						charStr := string(r)
						charWidth := lipgloss.Width(charStr)
						if currentWidth+charWidth > colWidths[i] {
							break
						}
						truncated += charStr
						currentWidth += charWidth
					}
					originalContent = truncated
				}
			}

			// Apply styling to the (possibly truncated) original content
			switch styleType {
			case "green":
				strVal = symbols.GreenText(originalContent)
			case "red":
				strVal = symbols.RedText(originalContent)
			case "blue":
				strVal = symbols.BlueText(originalContent)
			case "yellow":
				strVal = symbols.YellowText(originalContent)
			default:
				strVal = originalContent
			}

			// Manual padding calculation for styled strings
			// We need to pad based on visual width, not string length
			visualWidth := lipgloss.Width(strVal)
			paddingNeeded := colWidths[i] + 2 - visualWidth
			if paddingNeeded < 0 {
				paddingNeeded = 0
			}

			// Apply alternating row background color while preserving existing styling
			var cellContent string
			if isAlternateRow {
				if hasColor {
					// For colored text, we need to apply background to the content and padding separately
					// to preserve the existing foreground styling
					paddingStr := strings.Repeat(" ", paddingNeeded)
					styledPadding := lipgloss.NewStyle().Background(altRowBgColor).Render(paddingStr)

					// Apply background to the colored text by wrapping it with a background style
					styledContent := lipgloss.NewStyle().Background(altRowBgColor).Render(strVal)
					cellContent = styledContent + styledPadding
				} else {
					// For non-colored text, apply background to the entire cell content
					cellContent = strVal + strings.Repeat(" ", paddingNeeded)
					cellContent = lipgloss.NewStyle().Background(altRowBgColor).Render(cellContent)
				}
			} else {
				// No alternating background, just combine content with padding
				cellContent = strVal + strings.Repeat(" ", paddingNeeded)
			}

			output.WriteString(cellContent)
		}

		output.WriteString("\n")
	}

	return output.String()
}

// RenderCSV renders the data as CSV
func (m *BubbleTableModel) RenderCSV() string {
	// Reuse the existing CSV rendering logic
	printer := NewGenericTablePrinter(m.config, m.data)
	return printer.formatAsCSV()
}

// GetSelectedRow returns the currently selected row index (-1 if none)
func (m *BubbleTableModel) GetSelectedRow() int {
	return m.table.Cursor()
}

// GetSelectedData returns the data for the currently selected row
func (m *BubbleTableModel) GetSelectedData() (GenericTableData, bool) {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.data) {
		return m.data[cursor], true
	}
	return nil, false
}
