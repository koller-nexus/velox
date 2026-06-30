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

func (r *fakeRunner) Run(_ context.Context, s locate.Server, d *float64) speedtest.MeasurementResult {
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

func newApp(out, errw *bytes.Buffer, loc locate.Locator, run SpeedRunner, con ConsentManager) *App {
	return &App{
		Stdout:      out,
		Stderr:      errw,
		Locator:     loc,
		NewResolver: func(string) geo.Resolver { return fakeResolver{est: geo.LocationEstimate{Lat: -23.55, Lon: -46.63}} },
		Runner:      run,
		Consent:     con,
		LoadConfig:  func() (config.Config, error) { return config.Default(), nil },
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
