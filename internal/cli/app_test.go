package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/koller-nexus/velox/internal/config"
	"github.com/koller-nexus/velox/internal/geo"
	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/provider"
	"github.com/koller-nexus/velox/internal/speedtest"
)

type fakeLocator struct {
	servers []locate.Server
	err     error
}

func (f fakeLocator) Nearest(context.Context) ([]locate.Server, error) {
	return f.servers, f.err
}

type fakeResolver struct{ est geo.LocationEstimate }

func (f fakeResolver) Resolve(context.Context) (geo.LocationEstimate, error) { return f.est, nil }

type fakeConsent struct{ decision config.Decision }

func (f fakeConsent) Resolve(_, _ *os.File, _ io.Writer) (config.Decision, error) {
	return f.decision, nil
}
func (f fakeConsent) Decision() (config.Decision, error) { return f.decision, nil }
func (f fakeConsent) Set(config.Decision) error          { return nil }
func (f fakeConsent) Reset() error                       { return nil }

// fakeRunner echoes the server/distance it was given so the flow can be asserted.
type fakeRunner struct{ gotServer locate.Server }

func (r *fakeRunner) Run(_ context.Context, s locate.Server, d *float64, _ speedtest.Reporter) speedtest.MeasurementResult {
	r.gotServer = s
	return speedtest.MeasurementResult{
		Online:       true,
		DownloadMbps: 100,
		UploadMbps:   10,
		Server:       &speedtest.ServerInfo{Machine: s.Machine, IsFallback: s.IsFallback},
		DistanceKm:   d,
		PhaseStatus: map[speedtest.Phase]speedtest.PhaseOutcome{
			speedtest.PhaseDownload: {OK: true},
			speedtest.PhaseUpload:   {OK: true},
		},
	}
}

func (r *fakeRunner) Latency(_ context.Context, s locate.Server, d *float64) speedtest.LatencyResult {
	r.gotServer = s
	return speedtest.LatencyResult{
		Online:     true,
		LatencyMs:  9.9,
		JitterMs:   1.1,
		Server:     &speedtest.ServerInfo{Machine: s.Machine, IsFallback: s.IsFallback},
		DistanceKm: d,
	}
}

type countingResolver struct {
	est      geo.LocationEstimate
	called   int
	fallback error
}

func (c *countingResolver) Resolve(_ context.Context) (geo.LocationEstimate, error) {
	c.called++
	if c.fallback != nil {
		return geo.LocationEstimate{}, c.fallback
	}
	return c.est, nil
}

type recordingConsent struct {
	decision      config.Decision
	resolveCalled int
}

func (r *recordingConsent) Resolve(_, _ *os.File, _ io.Writer) (config.Decision, error) {
	r.resolveCalled++
	return r.decision, nil
}
func (r *recordingConsent) Decision() (config.Decision, error) { return r.decision, nil }
func (r *recordingConsent) Set(config.Decision) error          { return nil }
func (r *recordingConsent) Reset() error                       { return nil }

func newApp(out, errw *bytes.Buffer, loc locate.Locator, run SpeedRunner, con ConsentManager) *App {
	return &App{
		Stdout:         out,
		Stderr:         errw,
		StdinF:         os.Stdin,
		StdoutF:        os.Stdout,
		StderrF:        os.Stderr,
		Locator:        loc,
		NewResolver:    func(string) geo.Resolver { return fakeResolver{est: geo.LocationEstimate{Lat: -23.55, Lon: -46.63}} },
		Runner:         run,
		Consent:        con,
		LoadConfig:     func() (config.Config, error) { return config.Default(), nil },
		ProviderFinder: provider.NewFinder(),
	}
}

func TestVersionNoNetwork(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
	code := a.Run(context.Background(), []string{"--version"})
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.HasPrefix(out.String(), "velox ") {
		t.Errorf("version output = %q", out.String())
	}
}

func TestConsentGrantedSelectsNearest(t *testing.T) {
	servers := []locate.Server{
		{Machine: "london", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "saopaulo", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionGranted})

	code := a.Run(context.Background(), []string{"--check-internet"})
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if run.gotServer.Machine != "saopaulo" {
		t.Errorf("nearest = %q, want saopaulo", run.gotServer.Machine)
	}
}

func TestConsentDeniedUsesRegistryFirstNoDistance(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "second", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionDenied})

	code := a.Run(context.Background(), []string{"--check-internet", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if run.gotServer.Machine != "first" {
		t.Errorf("denied consent should keep registry order, got %q", run.gotServer.Machine)
	}
	var res speedtest.MeasurementResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if res.DistanceKm != nil {
		t.Errorf("distance must be null when consent denied")
	}
}

func TestLocateFailureFallsBackToAutoDiscovery(t *testing.T) {
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.DeadlineExceeded}, run, fakeConsent{decision: config.DecisionGranted})

	code := a.Run(context.Background(), []string{"--check-internet"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !run.gotServer.IsFallback {
		t.Errorf("expected fallback server on locate failure")
	}
}

func TestNoArgsIsUsageOnBareAndOKExit(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), nil); code != ExitOK {
		t.Errorf("bare invocation exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "USAGE") {
		t.Errorf("usage not printed")
	}
}

func TestServerOverrideBypassesSelection(t *testing.T) {
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, run, fakeConsent{})
	code := a.Run(context.Background(), []string{"--check-internet", "--server", "wss://h/ndt/v7/download"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if run.gotServer.DownloadURL != "wss://h/ndt/v7/download" {
		t.Errorf("override not applied: %+v", run.gotServer)
	}
}

func TestNoProgressFlagParsesAndRuns(t *testing.T) {
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, run, fakeConsent{})
	code := a.Run(context.Background(), []string{"--check-internet", "--no-progress"})
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestJSONOutputIsSingleValidDocument(t *testing.T) {
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, run, fakeConsent{})

	code := a.Run(context.Background(), []string{"--check-internet", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	// stdout must be exactly one valid JSON document, with no indicator escape codes.
	if strings.ContainsRune(out.String(), '\x1b') {
		t.Errorf("stdout contains escape sequences: %q", out.String())
	}
	var res speedtest.MeasurementResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("stdout is not a single valid JSON document: %v\n%s", err, out.String())
	}
}

func TestConsentGrantedIncludesNearestProvider(t *testing.T) {
	servers := []locate.Server{
		{Machine: "london", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "saopaulo", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionGranted})

	code := a.Run(context.Background(), []string{"--check-internet", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	var res speedtest.MeasurementResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if res.NearestProvider == nil {
		t.Fatal("expected nearest provider in result")
	}
}

func TestConsentDeniedNoLocationLookup(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "second", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	res := &countingResolver{est: geo.LocationEstimate{Lat: -23.55, Lon: -46.63}}
	a := &App{
		Stdout:         &out,
		Stderr:         &errw,
		StdinF:         os.Stdin,
		StdoutF:        os.Stdout,
		Locator:        fakeLocator{servers: servers},
		NewResolver:    func(string) geo.Resolver { return res },
		Runner:         run,
		Consent:        fakeConsent{decision: config.DecisionDenied},
		LoadConfig:     func() (config.Config, error) { return config.Default(), nil },
		ProviderFinder: provider.NewFinder(),
	}

	code := a.Run(context.Background(), []string{"--check-internet", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if res.called != 0 {
		t.Errorf("location lookup called %d times with denied consent", res.called)
	}
	var result speedtest.MeasurementResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json: %v", err)
	}
	if result.NearestProvider != nil {
		t.Error("nearest provider should be nil when consent denied")
	}
}

func TestNonInteractiveNoPromptAndNoLookup(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	res := &countingResolver{est: geo.LocationEstimate{Lat: -23.55, Lon: -46.63}}
	con := &recordingConsent{decision: config.DecisionUnset}
	a := &App{
		Stdout: &out,
		Stderr: &errw,
		// StdinF/StdoutF left nil => non-interactive.
		Locator:        fakeLocator{servers: servers},
		NewResolver:    func(string) geo.Resolver { return res },
		Runner:         run,
		Consent:        con,
		LoadConfig:     func() (config.Config, error) { return config.Default(), nil },
		ProviderFinder: provider.NewFinder(),
	}

	code := a.Run(context.Background(), []string{"--check-internet"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if con.resolveCalled != 0 {
		t.Error("consent.Resolve should not be called in non-interactive mode")
	}
	if res.called != 0 {
		t.Error("location lookup should not occur without prior consent in non-interactive mode")
	}
}

func TestLocationLookupFailureFallsBack(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	res := &countingResolver{fallback: context.DeadlineExceeded}
	a := &App{
		Stdout:         &out,
		Stderr:         &errw,
		StdinF:         os.Stdin,
		StdoutF:        os.Stdout,
		Locator:        fakeLocator{servers: servers},
		NewResolver:    func(string) geo.Resolver { return res },
		Runner:         run,
		Consent:        fakeConsent{decision: config.DecisionGranted},
		LoadConfig:     func() (config.Config, error) { return config.Default(), nil },
		ProviderFinder: provider.NewFinder(),
	}

	code := a.Run(context.Background(), []string{"--check-internet"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if res.called != 1 {
		t.Errorf("location lookup called %d times, want 1", res.called)
	}
	if run.gotServer.Machine != "first" {
		t.Errorf("fallback server = %q, want first", run.gotServer.Machine)
	}
}

func TestMissingCatalogFallsBackGracefully(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := &App{
		Stdout:         &out,
		Stderr:         &errw,
		StdinF:         os.Stdin,
		StdoutF:        os.Stdout,
		Locator:        fakeLocator{servers: servers},
		NewResolver:    func(string) geo.Resolver { return fakeResolver{est: geo.LocationEstimate{Lat: -23.55, Lon: -46.63}} },
		Runner:         run,
		Consent:        fakeConsent{decision: config.DecisionGranted},
		LoadConfig:     func() (config.Config, error) { return config.Default(), nil },
		ProviderFinder: &provider.Finder{Catalog: provider.Catalog{Version: 1, Providers: map[string]provider.Provider{}}},
	}

	code := a.Run(context.Background(), []string{"--check-internet", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	var res speedtest.MeasurementResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("json: %v", err)
	}
	if res.NearestProvider != nil {
		t.Error("nearest provider should be nil when catalog is empty")
	}
}

func TestNearestProviderFlagSelectsTarget(t *testing.T) {
	servers := []locate.Server{
		{Machine: "london", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "saopaulo", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionGranted})

	code := a.Run(context.Background(), []string{"--check-internet", "--nearest-provider"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if run.gotServer.Machine != "saopaulo" {
		t.Errorf("nearest-provider target = %q, want saopaulo", run.gotServer.Machine)
	}
	if !strings.Contains(out.String(), "Nearest provider:") {
		t.Errorf("output missing nearest provider line:\n%s", out.String())
	}
}

func TestNearestProviderFlagAbsentPreservesDefault(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "second", Lat: -23.43, Lon: -46.47, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionGranted})

	code := a.Run(context.Background(), []string{"--check-internet"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	// Default still ranks by user location (saopaulo is closest to fake -23.55,-46.63).
	if run.gotServer.Machine != "second" {
		t.Errorf("default server = %q, want second", run.gotServer.Machine)
	}
}

func TestNearestProviderFlagFallsBackWhenUnavailable(t *testing.T) {
	servers := []locate.Server{
		{Machine: "first", Lat: 51.47, Lon: -0.45, HasCoords: true},
	}
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, run, fakeConsent{decision: config.DecisionDenied})

	code := a.Run(context.Background(), []string{"--check-internet", "--nearest-provider"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if run.gotServer.Machine != "first" {
		t.Errorf("fallback server = %q, want first", run.gotServer.Machine)
	}
}
