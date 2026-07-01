// Package cli implements the velox command-line interface: flag parsing,
// command dispatch, the speed-test flow, and consent management.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/koller-nexus/velox/internal/config"
	"github.com/koller-nexus/velox/internal/consent"
	"github.com/koller-nexus/velox/internal/geo"
	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/ndt7"
	"github.com/koller-nexus/velox/internal/speedtest"
	"github.com/koller-nexus/velox/internal/version"
)

// Exit codes (contracts/cli-interface.md).
const (
	ExitOK      = 0
	ExitFailure = 1 // measurement/network failure
	ExitUsage   = 2 // bad flags/args
)

// SpeedRunner runs a measurement against a server. Implemented by *speedtest.Runner.
type SpeedRunner interface {
	Run(ctx context.Context, server locate.Server, distanceKm *float64, reporter speedtest.Reporter) speedtest.MeasurementResult
}

// ConsentManager gates and manages the location-consent decision. Implemented
// by *consent.Store; faked in tests.
type ConsentManager interface {
	Resolve(in, out *os.File, errw io.Writer) (config.Decision, error)
	Decision() (config.Decision, error)
	Set(config.Decision) error
	Reset() error
}

// App holds injectable dependencies so the flow can be tested without network.
type App struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   *os.File
	StdinF  *os.File // for TTY detection (usually == Stdin)
	StdoutF *os.File // for TTY detection
	StderrF *os.File // for TTY detection of the progress indicator's stream

	Locator     locate.Locator
	NewResolver func(endpoint string) geo.Resolver
	Runner      SpeedRunner
	Consent     ConsentManager
	LoadConfig  func() (config.Config, error)
}

// NewApp wires production dependencies.
func NewApp() *App {
	return &App{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Stdin:       os.Stdin,
		StdinF:      os.Stdin,
		StdoutF:     os.Stdout,
		StderrF:     os.Stderr,
		Locator:     locate.NewClient(),
		NewResolver: func(endpoint string) geo.Resolver { return geo.NewIPResolver(endpoint) },
		Runner:      speedtest.NewRunner(ndt7.NewMLabClient()),
		Consent:     consent.NewStore(),
		LoadConfig:  config.Load,
	}
}

// Run parses args and dispatches. It returns a process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printUsage()
		return ExitOK
	}
	if args[0] == "consent" {
		return a.runConsent(args[1:])
	}
	return a.runRoot(ctx, args)
}

func (a *App) runRoot(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("velox", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	var (
		check      = fs.Bool("check-internet", false, "run a full internet speed test")
		asJSON     = fs.Bool("json", false, "emit machine-readable JSON")
		server     = fs.String("server", "", "override the test server (ndt7 service URL)")
		timeout    = fs.Duration("timeout", 60*time.Second, "overall run budget")
		verbose    = fs.Bool("verbose", false, "verbose diagnostics on stderr")
		noProgress = fs.Bool("no-progress", false, "disable the loading indicator")
		showVer    = fs.Bool("version", false, "print version and exit")
		showHelp   = fs.Bool("help", false, "print help and exit")
	)
	fs.BoolVar(verbose, "v", false, "verbose diagnostics on stderr (shorthand)")

	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *showHelp {
		a.printUsage()
		return ExitOK
	}
	if *showVer {
		fmt.Fprintln(a.Stdout, version.String())
		return ExitOK
	}
	if !*check {
		a.printUsage()
		return ExitUsage
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	// Loading indicator: animates only on an interactive terminal and not under
	// --no-progress/--verbose. Writes to stderr; stopped before results render so
	// stdout stays clean (FR-003/FR-005/FR-006).
	ind := a.newIndicator(*noProgress, *verbose)
	ind.Start()
	defer ind.Stop() // safety net for cancellation / early-return paths (FR-007)

	ind.SetPhase("selecting server…")
	server2, distance := a.selectServer(ctx, *server, *verbose)
	if *verbose {
		fmt.Fprintf(a.Stderr, "velox: testing against %s\n", server2.Machine)
	}
	res := a.Runner.Run(ctx, server2, distance, phaseReporter{ind})
	ind.Stop() // clear the indicator before printing results (FR-006/SC-007)

	if *asJSON {
		if err := renderJSON(a.Stdout, res); err != nil {
			fmt.Fprintf(a.Stderr, "velox: render error: %v\n", err)
			return ExitFailure
		}
	} else {
		renderHuman(a.Stdout, res)
	}
	return exitCode(res)
}

// selectServer resolves which server to test against and the client distance.
// It applies the consent gate before any geolocation lookup (FR-004).
func (a *App) selectServer(ctx context.Context, override string, verbose bool) (locate.Server, *float64) {
	if override != "" {
		return locate.Server{
			Machine:     "(manual override)",
			DownloadURL: override,
			UploadURL:   override,
		}, nil
	}

	candidates, err := a.Locator.Nearest(ctx)
	if err != nil || len(candidates) == 0 {
		if verbose {
			fmt.Fprintf(a.Stderr, "velox: locate unavailable (%v); using auto-discovery\n", err)
		}
		return locate.Server{Machine: "(auto-discovered)", IsFallback: true}, nil
	}

	decision := a.resolveConsent()
	if decision != config.DecisionGranted {
		// No location resolution; use registry proximity ordering.
		sel, _ := geo.SelectNearest(candidates, nil)
		return sel.Server, sel.DistanceKm
	}

	endpoint := a.geoEndpoint()
	est, gerr := a.NewResolver(endpoint).Resolve(ctx)
	var estPtr *geo.LocationEstimate
	if gerr == nil {
		estPtr = &est
	} else if verbose {
		fmt.Fprintf(a.Stderr, "velox: geolocation failed (%v); ranking by registry order\n", gerr)
	}
	sel, _ := geo.SelectNearest(candidates, estPtr)
	return sel.Server, sel.DistanceKm
}

func (a *App) resolveConsent() config.Decision {
	d, err := a.Consent.Resolve(a.StdinF, a.StdoutF, a.Stderr)
	if err != nil {
		return config.DecisionDenied
	}
	return d
}

func (a *App) geoEndpoint() string {
	if a.LoadConfig == nil {
		return ""
	}
	c, err := a.LoadConfig()
	if err != nil {
		return ""
	}
	return c.GeoEndpoint
}

// exitCode maps a result to a process exit code (FR-001 AS#2/AS#4).
func exitCode(res speedtest.MeasurementResult) int {
	if !res.Online {
		return ExitFailure
	}
	for _, p := range []speedtest.Phase{speedtest.PhaseDownload, speedtest.PhaseUpload} {
		if o, ok := res.PhaseStatus[p]; ok && !o.OK {
			return ExitFailure
		}
	}
	return ExitOK
}
