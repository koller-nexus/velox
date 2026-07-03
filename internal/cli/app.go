// Package cli implements the velox command-line interface: flag parsing,
// command dispatch, the speed-test flow, and consent management.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/koller-nexus/velox/internal/config"
	"github.com/koller-nexus/velox/internal/consent"
	"github.com/koller-nexus/velox/internal/geo"
	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/ndt7"
	"github.com/koller-nexus/velox/internal/provider"
	"github.com/koller-nexus/velox/internal/speedtest"
	"github.com/koller-nexus/velox/internal/version"
)

// Exit codes (contracts/cli-interface.md).
const (
	ExitOK      = 0
	ExitFailure = 1 // measurement/network failure
	ExitUsage   = 2 // bad flags/args
)

// SpeedRunner runs measurements against a server. Implemented by *speedtest.Runner.
type SpeedRunner interface {
	Run(ctx context.Context, server locate.Server, distanceKm *float64, reporter speedtest.Reporter) speedtest.MeasurementResult
	Latency(ctx context.Context, server locate.Server, distanceKm *float64) speedtest.LatencyResult
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

	Locator        locate.Locator
	NewResolver    func(endpoint string) geo.Resolver
	Runner         SpeedRunner
	Consent        ConsentManager
	LoadConfig     func() (config.Config, error)
	ProviderFinder *provider.Finder

	location *geo.LocationEstimate // cached per invocation (FR-011)
}

// NewApp wires production dependencies.
func NewApp() *App {
	return &App{
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Stdin:          os.Stdin,
		StdinF:         os.Stdin,
		StdoutF:        os.Stdout,
		StderrF:        os.Stderr,
		Locator:        locate.NewClient(),
		NewResolver:    func(endpoint string) geo.Resolver { return geo.NewIPResolver(endpoint) },
		Runner:         speedtest.NewRunner(ndt7.NewMLabClient()),
		Consent:        consent.NewStore(),
		LoadConfig:     config.Load,
		ProviderFinder: provider.NewFinder(),
	}
}

// Run parses args and dispatches. It returns a process exit code.
//
// A bare invocation prints the overview. A first argument that is not a flag is
// treated as a subcommand and dispatched via the registry (FR-005/FR-007); a
// leading-dash first argument keeps the historical root flags
// (--check-internet/--version/--help).
func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printOverview()
		return ExitOK
	}
	if !strings.HasPrefix(args[0], "-") {
		return a.dispatch(ctx, args)
	}
	return a.runRoot(ctx, args)
}

func (a *App) runRoot(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("velox", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	var (
		check           = fs.Bool("check-internet", false, "run a full internet speed test")
		asJSON          = fs.Bool("json", false, "emit machine-readable JSON")
		server          = fs.String("server", "", "override the test server (ndt7 service URL)")
		nearestProvider = fs.Bool("nearest-provider", false, "prefer the M-Lab server nearest to your closest ISP POP")
		timeout         = fs.Duration("timeout", 60*time.Second, "overall run budget")
		verbose         = fs.Bool("verbose", false, "verbose diagnostics on stderr")
		noProgress      = fs.Bool("no-progress", false, "disable the loading indicator")
		showVer         = fs.Bool("version", false, "print version and exit")
		showHelp        = fs.Bool("help", false, "print help and exit")
	)
	fs.BoolVar(verbose, "v", false, "verbose diagnostics on stderr (shorthand)")
	fs.BoolVar(showHelp, "h", false, "print help and exit (shorthand)")

	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *showHelp {
		a.printOverview()
		return ExitOK
	}
	if *showVer {
		fmt.Fprintln(a.Stdout, version.String())
		return ExitOK
	}
	if !*check {
		a.printOverview()
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
	preferNearest := *nearestProvider || a.nearestProviderFromConfig()
	server2, distance, nearest := a.selectServer(ctx, *server, preferNearest, *verbose)
	if nearest != nil && nearest.SelectedServer != nil {
		server2.Machine = *nearest.SelectedServer
	}
	if *verbose {
		fmt.Fprintf(a.Stderr, "velox: testing against %s\n", server2.Machine)
	}
	res := a.Runner.Run(ctx, server2, distance, phaseReporter{ind})
	res.NearestProvider = nearest
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

func (a *App) nearestProviderFromConfig() bool {
	if a.LoadConfig == nil {
		return false
	}
	c, err := a.LoadConfig()
	if err != nil {
		return false
	}
	return c.NearestProvider
}

// selectServer resolves which server to test against, the client distance, and
// the nearest provider (when available). It applies the consent gate before any
// geolocation lookup (FR-004/FR-013).
func (a *App) selectServer(ctx context.Context, override string, preferNearest bool, verbose bool) (locate.Server, *float64, *provider.Result) {
	if override != "" {
		return locate.Server{
			Machine:     "(manual override)",
			DownloadURL: override,
			UploadURL:   override,
		}, nil, nil
	}

	candidates, err := a.Locator.Nearest(ctx)
	if err != nil || len(candidates) == 0 {
		if verbose {
			fmt.Fprintf(a.Stderr, "velox: locate unavailable (%v); using auto-discovery\n", err)
		}
		return locate.Server{Machine: "(auto-discovered)", IsFallback: true}, nil, nil
	}

	decision := a.resolveConsent()
	if decision != config.DecisionGranted {
		if verbose {
			fmt.Fprintln(a.Stderr, "velox: location consent not granted; skipping nearest-provider lookup")
		}
		sel, _ := geo.SelectNearest(candidates, nil)
		return sel.Server, sel.DistanceKm, nil
	}

	est, gerr := a.resolveLocation(ctx, verbose)
	var estPtr *geo.LocationEstimate
	if gerr == nil {
		estPtr = est
	} else if verbose {
		fmt.Fprintf(a.Stderr, "velox: geolocation failed (%v); ranking by registry order\n", gerr)
	}

	var nearest *provider.Result
	if estPtr != nil && a.ProviderFinder != nil {
		if r, ok := a.ProviderFinder.Nearest(*estPtr); ok {
			nearest = &r
			if preferNearest {
				srv, dist := provider.SelectServerForProvider(candidates, r)
				if srv.Machine != "" {
					selected := srv.Machine
					nearest.SelectedServer = &selected
					return srv, dist, nearest
				}
			}
		} else if verbose {
			fmt.Fprintln(a.Stderr, "velox: no nearest provider found in catalog")
		}
	}

	sel, _ := geo.SelectNearest(candidates, estPtr)
	return sel.Server, sel.DistanceKm, nearest
}

// resolveLocation returns the cached location estimate or resolves it once per
// invocation. It never writes the estimate to disk (FR-011/FR-012).
func (a *App) resolveLocation(ctx context.Context, _ bool) (*geo.LocationEstimate, error) {
	if a.location != nil {
		return a.location, nil
	}
	endpoint := a.geoEndpoint()
	est, err := a.NewResolver(endpoint).Resolve(ctx)
	if err != nil {
		return nil, err
	}
	a.location = &est
	return a.location, nil
}

// interactive reports whether velox should prompt for consent. A non-TTY stdin
// or non-TTY stdout means non-interactive execution (FR-013).
func (a *App) interactive() bool {
	return isTerminal(a.StdinF) && isTerminal(a.StdoutF)
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	// Avoid importing x/term solely for this check; use the consent package helper.
	return consent.IsTerminal(f)
}

func (a *App) resolveConsent() config.Decision {
	// In non-interactive mode, never prompt. Unset consent is treated as denied.
	if !a.interactive() {
		d, err := a.Consent.Decision()
		if err != nil || d == config.DecisionUnset {
			return config.DecisionDenied
		}
		return d
	}
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
