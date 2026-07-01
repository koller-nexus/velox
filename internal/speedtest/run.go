package speedtest

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/ndt7"
)

// DefaultFallbackHost is dialed for the connectivity check when no server host
// is known (e.g. auto-discovery path).
const DefaultFallbackHost = "locate.measurementlab.net:443"

// Reporter receives phase-transition notifications during a run. A nil Reporter
// is ignored. Calls are synchronous on the run goroutine, so implementations
// that touch shared state must be safe to call from it.
type Reporter interface {
	Phase(p Phase)
}

// report notifies r of a phase start, tolerating a nil Reporter.
func report(r Reporter, p Phase) {
	if r != nil {
		r.Phase(p)
	}
}

// Dialer abstracts connectivity probing so tests need no real network.
type Dialer func(ctx context.Context, address string) error

func tcpDial(ctx context.Context, address string) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return conn.Close()
}

// Runner performs a full measurement against a selected server.
type Runner struct {
	Client ndt7.Client
	Dial   Dialer // nil -> real TCP dial

	ConnectTimeout time.Duration // default 5s (SC-001)
	PhaseTimeout   time.Duration // default 20s per throughput phase
}

// NewRunner returns a Runner with default timeouts using real measurements.
func NewRunner(client ndt7.Client) *Runner {
	return &Runner{
		Client:         client,
		Dial:           tcpDial,
		ConnectTimeout: 5 * time.Second,
		PhaseTimeout:   20 * time.Second,
	}
}

// Run executes connectivity -> download (with latency) -> upload against the
// given server, honoring ctx cancellation. distanceKm is carried through to the
// result. If connectivity fails, throughput phases are skipped and Online is
// false (FR-001). reporter, if non-nil, is notified as each phase begins
// (presentation only; it must not affect the result — FR-008/FR-010).
func (r *Runner) Run(ctx context.Context, server locate.Server, distanceKm *float64, reporter Reporter) (res MeasurementResult) {
	res = MeasurementResult{
		StartedAt:   time.Now().UTC(),
		DistanceKm:  distanceKm,
		PhaseStatus: map[Phase]PhaseOutcome{},
		Server: &ServerInfo{
			Machine:    server.Machine,
			City:       server.City,
			Country:    server.Country,
			IsFallback: server.IsFallback,
		},
	}
	start := time.Now()
	defer func() { res.DurationMs = time.Since(start).Milliseconds() }()

	// Phase 1: connectivity.
	report(reporter, PhaseConnectivity)
	if err := r.connectivity(ctx, server); err != nil {
		res.Online = false
		res.PhaseStatus[PhaseConnectivity] = PhaseOutcome{OK: false, Error: err.Error()}
		return res
	}
	res.Online = true
	res.PhaseStatus[PhaseConnectivity] = PhaseOutcome{OK: true}

	// Phase 2+3: download (also yields latency/jitter).
	report(reporter, PhaseDownload)
	dl, err := r.runPhase(ctx, func(c context.Context) (ndt7.Throughput, error) {
		return r.Client.Download(c, server.DownloadURL)
	})
	if err != nil {
		res.PhaseStatus[PhaseDownload] = PhaseOutcome{OK: false, Error: err.Error()}
		res.PhaseStatus[PhaseLatency] = PhaseOutcome{OK: false, Error: err.Error()}
	} else {
		res.DownloadMbps = dl.Mbps
		res.LatencyMs = dl.MinRTTMs
		res.JitterMs = dl.JitterMs
		res.PhaseStatus[PhaseDownload] = PhaseOutcome{OK: true, Value: dl.Mbps}
		res.PhaseStatus[PhaseLatency] = PhaseOutcome{OK: true, Value: dl.MinRTTMs}
	}

	// Phase 4: upload.
	report(reporter, PhaseUpload)
	ul, err := r.runPhase(ctx, func(c context.Context) (ndt7.Throughput, error) {
		return r.Client.Upload(c, server.UploadURL)
	})
	if err != nil {
		res.PhaseStatus[PhaseUpload] = PhaseOutcome{OK: false, Error: err.Error()}
	} else {
		res.UploadMbps = ul.Mbps
		res.PhaseStatus[PhaseUpload] = PhaseOutcome{OK: true, Value: ul.Mbps}
	}

	return res
}

func (r *Runner) connectivity(ctx context.Context, server locate.Server) error {
	addr := hostPort(server.DownloadURL)
	if addr == "" {
		addr = DefaultFallbackHost
	}
	dial := r.Dial
	if dial == nil {
		dial = tcpDial
	}
	cctx, cancel := context.WithTimeout(ctx, r.connectTimeout())
	defer cancel()
	return dial(cctx, addr)
}

func (r *Runner) runPhase(ctx context.Context, fn func(context.Context) (ndt7.Throughput, error)) (ndt7.Throughput, error) {
	pctx, cancel := context.WithTimeout(ctx, r.phaseTimeout())
	defer cancel()
	return fn(pctx)
}

func (r *Runner) connectTimeout() time.Duration {
	if r.ConnectTimeout <= 0 {
		return 5 * time.Second
	}
	return r.ConnectTimeout
}

func (r *Runner) phaseTimeout() time.Duration {
	if r.PhaseTimeout <= 0 {
		return 20 * time.Second
	}
	return r.PhaseTimeout
}

// hostPort extracts host:port from a wss:// URL, defaulting to port 443.
func hostPort(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Port() != "" {
		return u.Host
	}
	return net.JoinHostPort(u.Hostname(), "443")
}
