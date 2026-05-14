/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// resetAPCmd is `wifimgr reset ap <ap-name> [site <site-name>] [force]`.
//
// The site keyword is optional; when supplied it acts as a guardrail — the
// command refuses to run if the named AP isn't actually in that site. `force`
// skips the y/N prompt for scripted use.
var resetAPCmd = &cobra.Command{
	Use:   "ap <ap-name> [site <site-name>] [force]",
	Short: "Reboot a single access point",
	Long: `Reboot one access point via its owning vendor API.

The AP is looked up in the local cache by name; the owning API and site are
inferred from that record. Pass 'site <name>' to assert the expected site as
a guardrail. Pass 'force' to skip the y/N confirmation prompt.

Vendor support:
  - Mist:     POST /api/v1/sites/{site}/devices/{device}/restart
  - Meraki:   POST /devices/{serial}/reboot
  - Ubiquiti: not supported (Site Manager API is read-only) — the command
              prints "This feature is not available with this API
              (<label>:<vendor>)." and exits non-zero.`,
	Example: `  wifimgr reset ap AP-LAB-01
  wifimgr reset ap AP-LAB-01 site US-LAB-01
  wifimgr reset ap AP-LAB-01 site US-LAB-01 force`,
	RunE: runResetAP,
}

func init() {
	resetCmd.AddCommand(resetAPCmd)
}

func runResetAP(cmd *cobra.Command, args []string) error {
	for _, a := range args {
		if strings.ToLower(a) == "help" {
			return cmd.Help()
		}
	}

	parsed, err := cmdutils.ParseResetArgs(args)
	if err != nil {
		return err
	}

	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		return fmt.Errorf("failed to get cache accessor: %w", err)
	}

	device, err := cacheAccessor.GetDeviceByName(parsed.APName)
	if err != nil {
		return fmt.Errorf("AP %q not found: %w (try: wifimgr refresh device)", parsed.APName, err)
	}
	if device.Type != "ap" {
		return fmt.Errorf("device %q is not an AP (type: %s)", parsed.APName, device.Type)
	}

	if parsed.SiteName != "" {
		expected, err := cacheAccessor.GetSiteByID(device.SiteID)
		if err != nil {
			return fmt.Errorf("AP %q has site_id %q which is not in the cache: %w",
				parsed.APName, device.SiteID, err)
		}
		if !strings.EqualFold(expected.Name, parsed.SiteName) {
			return fmt.Errorf("AP %q is in site %q, not %q",
				parsed.APName, expected.Name, parsed.SiteName)
		}
	}

	apiLabel := device.SourceAPI
	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	client, err := registry.GetClient(apiLabel)
	if err != nil {
		return fmt.Errorf("failed to get client for %s: %w", apiLabel, err)
	}
	vendor, _ := registry.GetVendor(apiLabel)

	siteLabel := device.SiteName
	if siteLabel == "" {
		siteLabel = device.SiteID
	}

	if !parsed.Force {
		ok, err := confirmReboot(parsed.APName, siteLabel, apiLabel, vendor)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Printf("Rebooting AP %s at %s via %s (%s)…\n",
		parsed.APName, siteLabel, apiLabel, vendor)

	if err := client.Devices().Reboot(globalContext, device.SiteID, device.ID); err != nil {
		return renderResetError(err, apiLabel, vendor)
	}

	fmt.Printf("Reboot request accepted for %s at %s (%s:%s). AP will restart in a few seconds.\n",
		parsed.APName, siteLabel, apiLabel, vendor)
	return nil
}

// confirmReboot prompts on stdin for y/N. On a non-tty stdin it refuses by
// default — scripted callers must pass `force` rather than relying on stdin.
func confirmReboot(apName, siteLabel, apiLabel, vendor string) (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, fmt.Errorf("stdin is not a terminal — pass 'force' to skip confirmation")
	}

	fmt.Printf("Reboot AP %q at site %q via %s (%s)? [y/N] ", apName, siteLabel, apiLabel, vendor)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

// renderResetError converts an error from the vendor layer into a
// user-friendly message. CapabilityNotSupportedError is rendered in the
// exact phrase requested ("This feature is not available with this API
// (<label>:<vendor>).") regardless of any APILabel/VendorName fields the
// adapter may have left unset.
//
// Vendor adapters (notably meraki) embed the orgID as APILabel because the
// service doesn't know the registry label at construction time. We rewrite
// APILabel to the registry label here so the user sees the API they
// configured ("meraki"), not the opaque orgID. Also normalises a few
// common low-level failures (DNS, TCP timeout, generic network errors)
// into a friendly one-liner — the raw error is preserved for `-d` debug.
func renderResetError(err error, apiLabel, vendor string) error {
	var capErr *vendors.CapabilityNotSupportedError
	if errors.As(err, &capErr) {
		return fmt.Errorf("this feature is not available with this API (%s:%s)", apiLabel, vendor)
	}

	var authErr *vendors.AuthError
	if errors.As(err, &authErr) {
		authErr.APILabel = apiLabel
		return fmt.Errorf("%s", authErr.UserMessage())
	}
	var rlErr *vendors.RateLimitError
	if errors.As(err, &rlErr) {
		rlErr.APILabel = apiLabel
		return fmt.Errorf("%s", rlErr.UserMessage())
	}
	var srvErr *vendors.ServerError
	if errors.As(err, &srvErr) {
		srvErr.APILabel = apiLabel
		return fmt.Errorf("%s", srvErr.UserMessage())
	}
	var nfErr *vendors.NotFoundError
	if errors.As(err, &nfErr) {
		nfErr.APILabel = apiLabel
		return fmt.Errorf("%s", nfErr.UserMessage())
	}
	var tErr *vendors.TransportError
	if errors.As(err, &tErr) {
		tErr.APILabel = apiLabel
		return renderTransportError(tErr, apiLabel, vendor)
	}
	return err
}

// renderTransportError gives DNS / TCP / TLS timeout failures a short,
// actionable user-facing message instead of dumping the raw resty/SDK error.
// Status==0 means the request never reached a response — almost always a
// local connectivity problem.
func renderTransportError(tErr *vendors.TransportError, apiLabel, vendor string) error {
	if tErr.Status != 0 {
		return fmt.Errorf("%s", tErr.UserMessage())
	}

	raw := ""
	if tErr.Err != nil {
		raw = tErr.Err.Error()
	}
	switch {
	case strings.Contains(raw, "i/o timeout"),
		strings.Contains(raw, "deadline exceeded"):
		return fmt.Errorf("%s (%s): connection timed out reaching the vendor API. Check your network and retry.",
			apiLabel, vendor)
	case strings.Contains(raw, "no such host"),
		strings.Contains(raw, "lookup"):
		return fmt.Errorf("%s (%s): DNS lookup failed for the vendor API endpoint. Check your network and retry.",
			apiLabel, vendor)
	case strings.Contains(raw, "connection refused"),
		strings.Contains(raw, "connection reset"):
		return fmt.Errorf("%s (%s): the vendor API refused or dropped the connection. Check your network and retry.",
			apiLabel, vendor)
	case strings.Contains(raw, "SDK panicked"):
		return fmt.Errorf("%s (%s): the vendor SDK crashed during the request — likely a transient connectivity issue. Retry; if it persists, run with -d for details.",
			apiLabel, vendor)
	}
	return fmt.Errorf("%s", tErr.UserMessage())
}
