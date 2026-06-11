package cmdutils

import (
	"fmt"
	"os"
)

// Operational runtime state, set once from persistent flags at startup. These
// govern HOW the app runs — output verbosity and confirmation behavior — which
// is the proper job of flags. The positional grammar is reserved for WHAT the
// app does (the domain interaction). Kept here, in a low-level package, so both
// the cmd and cmd/apply packages can consult them without an import cycle.
var (
	quiet     bool
	assumeYes bool
	noInput   bool
)

// SetQuiet records the --quiet flag.
func SetQuiet(v bool) { quiet = v }

// SetAssumeYes records the --yes flag.
func SetAssumeYes(v bool) { assumeYes = v }

// SetNoInput records the --no-input flag.
func SetNoInput(v bool) { noInput = v }

// Quiet reports whether non-essential output should be suppressed.
func Quiet() bool { return quiet }

// AssumeYes reports whether confirmations should be auto-approved.
func AssumeYes() bool { return assumeYes }

// NoInput reports whether prompting is forbidden (fail closed instead).
func NoInput() bool { return noInput }

// Noticef writes a non-essential status line ("Armed 5 devices", "Wrote import
// file …") to stderr, unless --quiet is set. Notices are operational feedback,
// not primary output, so they stay on stderr to keep piped stdout clean.
func Noticef(format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
