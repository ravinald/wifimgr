package utils

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/internal/common"
)

// PromptForConfirmation asks the user for confirmation and returns true if confirmed
func PromptForConfirmation(message string) bool {
	var confirm string
	fmt.Print(message)
	_, err := fmt.Scanln(&confirm)
	if err != nil {
		return false
	}

	confirm = strings.ToLower(confirm)
	return confirm == "y" || confirm == "yes"
}

// MaskString masks sensitive information like API tokens
// This is a wrapper around common.MaskString for backward compatibility
func MaskString(s string) string {
	return common.MaskString(s)
}

// FormatOutputWithWarning returns the text unchanged.
// Legacy cache integrity warning system has been removed.
func FormatOutputWithWarning(text string) string {
	return text
}

// PrintWithWarning prints a line of text formatted as a section heading in blue.
// Legacy cache integrity warning system has been removed.
func PrintWithWarning(format string, args ...interface{}) {
	var text string
	if len(args) > 0 {
		text = fmt.Sprintf(format, args...)
	} else {
		text = format
	}
	blueText := color.New(color.FgBlue, color.Bold).Sprint(text)
	fmt.Println(blueText)
}

// PrintDetailWithWarning prints a detail line (like ID, Name, etc.) without applying blue color.
// Legacy cache integrity warning system has been removed.
func PrintDetailWithWarning(format string, args ...interface{}) {
	var text string
	if len(args) > 0 {
		text = fmt.Sprintf(format, args...)
	} else {
		text = format
	}
	fmt.Println(text)
}

// PrintTextWithWarning prints a text string formatted as a section heading in blue.
// Legacy cache integrity warning system has been removed.
func PrintTextWithWarning(text string) {
	blueText := color.New(color.FgBlue, color.Bold).Sprint(text)
	fmt.Println(blueText)
}
