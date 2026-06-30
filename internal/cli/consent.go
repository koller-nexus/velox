package cli

import (
	"flag"
	"fmt"

	"github.com/koller-nexus/velox/internal/config"
)

// runConsent implements `velox consent` (FR-005/006).
func (a *App) runConsent(args []string) int {
	fs := flag.NewFlagSet("velox consent", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	var (
		status = fs.Bool("status", false, "print the current consent decision")
		reset  = fs.Bool("reset", false, "clear the stored consent decision")
		grant  = fs.Bool("grant", false, "record consent as granted")
		deny   = fs.Bool("deny", false, "record consent as denied")
	)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
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
		fmt.Fprintln(a.Stderr, "velox consent: specify one of --status, --reset, --grant, --deny")
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
