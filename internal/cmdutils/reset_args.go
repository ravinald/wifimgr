/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmdutils

import (
	"fmt"
	"strings"
)

// ResetArgs holds the parsed positional arguments for `reset ap`.
type ResetArgs struct {
	APName   string // required: AP name (or MAC) to reset
	SiteName string // optional: site filter via `site <name>`
	Force    bool   // optional: trailing `force` keyword skips confirmation
}

// ParseResetArgs parses positional args for `reset ap`.
//
// Recognised forms:
//
//	<ap-name>
//	<ap-name> site <site-name>
//	<ap-name> force
//	<ap-name> site <site-name> force
//	<ap-name> force site <site-name>      # any keyword order after the name
//
// The first non-keyword token is the AP name. `site <name>` and `force` may
// appear in either order. Anything else is rejected with a hint so users
// don't accidentally pass flags or unrecognized tokens.
func ParseResetArgs(args []string) (*ResetArgs, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing AP name (usage: reset ap <ap-name> [site <site-name>] [force])")
	}

	result := &ResetArgs{
		APName: StripQuotes(args[0]),
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch strings.ToLower(arg) {
		case "site":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'site' requires a site name")
			}
			if result.SiteName != "" {
				return nil, fmt.Errorf("site specified multiple times")
			}
			result.SiteName = StripQuotes(args[i+1])
			i++

		case "force":
			if result.Force {
				return nil, fmt.Errorf("'force' specified multiple times")
			}
			result.Force = true

		default:
			return nil, fmt.Errorf("unexpected positional %q (expected 'site <name>' or 'force')", arg)
		}
	}

	return result, nil
}
