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

// Refresh scope levels. Default (empty) is the managed set with no client
// detail; "detail" adds per-client detail; "all" drops the managed filter and
// fetches everything the API has (and client detail too).
const (
	RefreshScopeManaged = ""
	RefreshScopeDetail  = "detail"
	RefreshScopeAll     = "all"
)

// RefreshArgs holds the parsed positional arguments for refresh commands.
type RefreshArgs struct {
	Target   string // set by `target <api-label>`
	SiteName string // set by `site <site-name>` (or, with AllowImplicitSite, the leading bare token)
	Scope    string // "", "detail", or "all"
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

	// AllowSite: if false, `site <name>` is rejected.
	AllowSite bool

	// AllowScope: if false, the `detail`/`all` scope words are rejected. Used
	// by `refresh client site`, whose scope is fixed.
	AllowScope bool
}

// ParseRefreshArgs parses positional arguments for refresh commands.
//
// Recognised forms (with AllowSite and AllowScope true):
//
//	[]                                  managed set, every API
//	all                                 everything the API has, every API
//	detail                              managed set + client detail
//	site <site-name>                    managed set for one site
//	site <site-name> all                everything the API has for one site
//	site <site-name> detail             managed set + client detail for one site
//	... target <api-label>              disambiguate a site spanning APIs
//
// `api <api-label>` is rejected with a migration hint pointing to
// `target <api-label>`.
func ParseRefreshArgs(args []string, opts ParseRefreshOptions) (*RefreshArgs, error) {
	result := &RefreshArgs{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch strings.ToLower(arg) {
		case "target":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'target' requires an API label")
			}
			if result.Target != "" {
				return nil, fmt.Errorf("target specified multiple times")
			}
			result.Target = StripQuotes(args[i+1])
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
			result.SiteName = StripQuotes(args[i+1])
			i++

		case RefreshScopeDetail, RefreshScopeAll:
			if !opts.AllowScope {
				return nil, fmt.Errorf("%q is not valid here", arg)
			}
			if result.Scope != "" {
				return nil, fmt.Errorf("scope specified multiple times (have %q)", result.Scope)
			}
			result.Scope = strings.ToLower(arg)

		case "api":
			return nil, fmt.Errorf("the 'api' keyword has been removed; use 'target <api-label>' instead")

		default:
			// At i==0 with AllowImplicitSite, the bare token is the site name.
			if i == 0 && opts.AllowImplicitSite && opts.AllowSite {
				result.SiteName = StripQuotes(arg)
				continue
			}
			return nil, fmt.Errorf("unexpected positional %q", arg)
		}
	}

	return result, nil
}
