package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/koller-nexus/velox/internal/config"
)

const consentUsage = `velox consent — manage the location-lookup consent decision.

USAGE:
  velox consent --status    Print the current decision (granted|denied|unset).
  velox consent --grant     Allow IP-based location lookups.
  velox consent --deny      Disallow location lookups.
  velox consent --reset     Forget the decision (asked again next time).

velox asks for consent once before using your approximate, IP-based location to
pick the nearest server. See 'velox --help' for the privacy summary.
`

// consentCommand registers `velox consent` in the command registry (FR-005/FR-007)
// so it appears in `velox help` and supports `velox consent --help`.
func (a *App) consentCommand() *Command {
	return &Command{
		Name:    "consent",
		Summary: "Manage location-lookup consent (status/grant/deny/reset)",
		Usage:   consentUsage,
		Run:     a.runConsent,
	}
}

// runConsent implements `velox consent` (FR-005/006).
func (a *App) runConsent(_ context.Context, args []string) int {
	fs := flag.NewFlagSet("consent", flag.ContinueOnError)
	var (
		status = fs.Bool("status", false, "print the current consent decision")
		reset  = fs.Bool("reset", false, "clear the stored consent decision")
		grant  = fs.Bool("grant", false, "record consent as granted")
		deny   = fs.Bool("deny", false, "record consent as denied")
	)
	if code, handled := a.parseCommandFlags(fs, consentUsage, args); handled {
		return code
	}

	switch {
	case *status:
		d, err := a.Consent.Decision()
		if err != nil {
			fmt.Fprintf(a.Stderr, "velox: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintln(a.Stdout, d)
		return ExitOK
	case *reset:
		if err := a.Consent.Reset(); err != nil {
			fmt.Fprintf(a.Stderr, "velox: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintln(a.Stderr, "velox: consent reset; you will be asked again next time location is needed")
		return ExitOK
	case *grant:
		return a.setConsent(config.DecisionGranted)
	case *deny:
		return a.setConsent(config.DecisionDenied)
	default:
		fmt.Fprintln(a.Stderr, "velox consent: specify one of --status, --reset, --grant, --deny (see 'velox help consent')")
		return ExitUsage
	}
}

func (a *App) setConsent(d config.Decision) int {
	if err := a.Consent.Set(d); err != nil {
		fmt.Fprintf(a.Stderr, "velox: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(a.Stdout, "consent %s\n", d)
	return ExitOK
}
