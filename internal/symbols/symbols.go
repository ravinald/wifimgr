package symbols

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// ConfigureColor applies the color policy once at startup. Styled output is
// disabled — globally, for every lipgloss-rendered string — when any of these
// hold: an explicit --no-color, NO_COLOR set (any value), TERM=dumb, or stdout
// is not a TTY. Structured formats (json/csv) are plain by construction and
// don't depend on this; it governs the human table/symbol output.
func ConfigureColor(noColorFlag bool) {
	stdoutFd := int(os.Stdout.Fd()) // #nosec G115 -- file descriptors are small non-negative integers
	disabled := noColorFlag ||
		os.Getenv("NO_COLOR") != "" ||
		os.Getenv("TERM") == "dumb" ||
		!term.IsTerminal(stdoutFd)
	if disabled {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

var (
	// greenStyle creates a bold green style
	greenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	// redStyle creates a bold red style
	redStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	// blueStyle creates a bold blue style
	blueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0080FF")).
			Bold(true)

	// yellowStyle creates a bold yellow style
	yellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)

	// emphasisStyle is a theme-safe high-contrast emphasis: bold with an
	// adaptive foreground (near-black on light terminals, near-white on dark).
	// It carries no operational meaning — color is reserved for device state —
	// so it marks metadata like managed rows and legend keys without implying up.
	emphasisStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}).
			Bold(true)
)

// isTerminal checks if we're running in a terminal that supports colors and symbols
func isTerminal() bool {
	// Check file descriptors to determine if we're in a terminal
	stdoutFd := int(os.Stdout.Fd()) // #nosec G115 -- file descriptors are small non-negative integers
	stdinFd := int(os.Stdin.Fd())   // #nosec G115 -- file descriptors are small non-negative integers
	stderrFd := int(os.Stderr.Fd()) // #nosec G115 -- file descriptors are small non-negative integers

	// Try stdout first (most common case for CLI output)
	if term.IsTerminal(stdoutFd) {
		return true
	}

	// Fallback to stdin (interactive terminal)
	if term.IsTerminal(stdinFd) {
		return true
	}

	// Fallback to stderr (error output terminal)
	if term.IsTerminal(stderrFd) {
		return true
	}

	return false
}

// Status prefix functions for user-facing messages

// SuccessPrefix returns a colored success indicator for status messages.
func SuccessPrefix() string {
	return GreenText("[OK]")
}

// FailurePrefix returns a colored failure indicator for status messages.
func FailurePrefix() string {
	return RedText("[FAIL]")
}

// ErrorPrefix returns a colored error indicator for status messages.
func ErrorPrefix() string {
	return RedText("[ERROR]")
}

// WarningPrefix returns a colored warning indicator for status messages.
func WarningPrefix() string {
	return YellowText("[WARN]")
}

// FormatBooleanValue formats a boolean value based on whether it's a connection field
func FormatBooleanValue(value bool, isConnectionField bool) string {
	if isConnectionField {
		// Use colored C/D for connection fields
		if value {
			return GreenText("C")
		} else {
			return RedText("D")
		}
	} else {
		// Use plain Yes/No for non-connection boolean fields
		if value {
			return "Yes"
		} else {
			return "No"
		}
	}
}

// GreenText returns green colored text or plain text for non-terminals
func GreenText(text string) string {
	if isTerminal() {
		return greenStyle.Render(text)
	}
	return text
}

// RedText returns red colored text or plain text for non-terminals
func RedText(text string) string {
	if isTerminal() {
		return redStyle.Render(text)
	}
	return text
}

// BlueText returns blue colored text or plain text for non-terminals
func BlueText(text string) string {
	if isTerminal() {
		return blueStyle.Render(text)
	}
	return text
}

// YellowText returns yellow colored text or plain text for non-terminals
func YellowText(text string) string {
	if isTerminal() {
		return yellowStyle.Render(text)
	}
	return text
}

// EmphasisText returns theme-safe bold-emphasis text, or plain text for
// non-terminals. Used for metadata emphasis (managed rows, legend keys) where
// color would wrongly read as device state.
func EmphasisText(text string) string {
	if isTerminal() {
		return emphasisStyle.Render(text)
	}
	return text
}
