package cmdutils

import (
	"fmt"
	"strings"
)

// ApplyOptions carries the optional positional flags that may appear after the
// required positional arguments of an apply subcommand
// (`diff`, `split`, `no-refresh`, `force`).
type ApplyOptions struct {
	DiffMode  bool
	SplitDiff bool
	NoRefresh bool
	Force     bool
}

// validApplyOptions enumerates the legal optional tokens for apply commands.
var validApplyOptions = map[string]bool{
	"diff":       true,
	"split":      true,
	"no-refresh": true,
	"force":      true,
}

// ParseApplyOptions reads the optional positional tokens from args.
// Unknown tokens are silently ignored — pair with ValidateApplyOptions
// in the command's Args validator if strict checking is desired.
func ParseApplyOptions(args []string) ApplyOptions {
	var opts ApplyOptions
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "diff":
			opts.DiffMode = true
		case "split":
			opts.SplitDiff = true
		case "no-refresh":
			opts.NoRefresh = true
		case "force":
			opts.Force = true
		}
	}
	return opts
}

// ValidateApplyOptions returns an error if any token in args is not one of the
// legal apply options. Intended for use inside cobra Args validators after
// the required positional arguments have been verified.
func ValidateApplyOptions(args []string) error {
	for _, arg := range args {
		if !validApplyOptions[strings.ToLower(arg)] {
			return fmt.Errorf("unexpected argument: %s (valid options: diff, split, no-refresh, force)", arg)
		}
	}
	return nil
}
