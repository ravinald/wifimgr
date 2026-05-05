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

// RefreshArgs holds the parsed positional arguments for refresh commands.
type RefreshArgs struct {
	APIName  string // set by `api <api-label>`
	SiteName string // set by `site <site-name>` (or, with AllowImplicitSite, the leading bare token)
}

// ParseRefreshOptions tunes ParseRefreshArgs for callers whose cobra subcommand
// path already establishes context (e.g. `refresh client site` consumes the
// `site` keyword as a subcommand name, so the args entering the parser start
// with a bare site name).
type ParseRefreshOptions struct {
	// AllowImplicitSite: if true, a leading non-keyword token is treated as
	// the site name. Used by `refresh client site <name> ...` where cobra has
	// already consumed the `site` keyword.
	AllowImplicitSite bool

	// AllowSite: if false, `site <name>` is rejected. Used to constrain
	// commands that don't support per-site refresh.
	AllowSite bool
}

// ParseRefreshArgs parses positional arguments for refresh commands.
//
// Recognised forms (when AllowSite is true):
//
//	[]
//	api <api-label>
//	site <site-name>
//	site <site-name> api <api-label>
//	api <api-label> site <site-name>
//
// `target <api-label>` is rejected with an explicit migration error pointing
// to `api <api-label>`. A bare leading token (e.g. `<api-label>` without the
// `api` keyword) is rejected with a migration hint, unless
// AllowImplicitSite=true in which case the leading token is interpreted as
// the site name.
func ParseRefreshArgs(args []string, opts ParseRefreshOptions) (*RefreshArgs, error) {
	result := &RefreshArgs{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch strings.ToLower(arg) {
		case "api":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'api' requires an API label")
			}
			if result.APIName != "" {
				return nil, fmt.Errorf("api specified multiple times")
			}
			result.APIName = stripQuotes(args[i+1])
			i++

		case "site":
			if !opts.AllowSite {
				return nil, fmt.Errorf("'site' is not valid here")
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'site' requires a site name")
			}
			if result.SiteName != "" {
				return nil, fmt.Errorf("site specified multiple times")
			}
			result.SiteName = stripQuotes(args[i+1])
			i++

		case "target":
			return nil, fmt.Errorf("the 'target' keyword has been removed; use 'api <api-label>' instead")

		default:
			// At i==0 with AllowImplicitSite, the bare token is the site name.
			if i == 0 && opts.AllowImplicitSite && opts.AllowSite {
				result.SiteName = stripQuotes(arg)
				continue
			}
			// Otherwise this is a migration error: the old `refresh device <api>`
			// shape is gone. Give a precise hint pointing to the new form.
			return nil, fmt.Errorf("unexpected positional %q — did you mean 'api %s'?", arg, arg)
		}
	}

	return result, nil
}
