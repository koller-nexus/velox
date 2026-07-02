// Package ndt7 wraps the canonical M-Lab ndt7 client behind a small interface
// so the rest of velox can be tested without the network. ndt7 runs over
// WebSocket-over-TLS (wss), needing no elevated privileges (FR-018).
package ndt7

import (
	"context"
	"fmt"
	"math"
	"net/url"

	"github.com/koller-nexus/velox/internal/version"
	ndt7lib "github.com/m-lab/ndt7-client-go"
	"github.com/m-lab/ndt7-client-go/spec"
)

// Throughput is the result of a single ndt7 measurement phase.
type Throughput struct {
	Mbps     float64 // mean application-layer goodput, megabits per second
	MinRTTMs float64 // minimum round-trip time in milliseconds (download only)
	JitterMs float64 // RTT variation in milliseconds (download only)
}

// Client measures download/upload throughput against an ndt7 server.
// An empty serviceURL lets the underlying client auto-discover a server via the
// M-Lab Locate API (used as the fallback path when location is unavailable).
type Client interface {
	Download(ctx context.Context, serviceURL string) (Throughput, error)
	Upload(ctx context.Context, serviceURL string) (Throughput, error)
	// DownloadLatency runs the download subtest to sample RTT/jitter for a
	// latency-only check. Unlike Download, when ctx's deadline fires it returns
	// the RTT statistics gathered so far instead of an error, so a short
	// sampling window still yields a reading (see internal/speedtest.Runner.Latency).
	DownloadLatency(ctx context.Context, serviceURL string) (Throughput, error)
}

// MLabClient is the production Client backed by github.com/m-lab/ndt7-client-go.
type MLabClient struct{}

// NewMLabClient returns a Client that performs real ndt7 measurements.
func NewMLabClient() *MLabClient { return &MLabClient{} }

func newLibClient(serviceURL string) (*ndt7lib.Client, error) {
	c := ndt7lib.NewClient("velox", version.Version)
	if serviceURL != "" {
		u, err := url.Parse(serviceURL)
		if err != nil {
			return nil, fmt.Errorf("parse service url: %w", err)
		}
		c.ServiceURL = u
	}
	return c, nil
}

// Download measures download goodput and minimum RTT.
func (m *MLabClient) Download(ctx context.Context, serviceURL string) (Throughput, error) {
	c, err := newLibClient(serviceURL)
	if err != nil {
		return Throughput{}, err
	}
	ch, err := c.StartDownload(ctx)
	if err != nil {
		return Throughput{}, fmt.Errorf("start ndt7 download: %w", err)
	}
	return consume(ctx, ch, true, false)
}

// DownloadLatency runs the download subtest but returns whatever RTT statistics
// were gathered when the context deadline fires, so a short window still yields
// a latency/jitter reading (no throughput guarantees).
func (m *MLabClient) DownloadLatency(ctx context.Context, serviceURL string) (Throughput, error) {
	c, err := newLibClient(serviceURL)
	if err != nil {
		return Throughput{}, err
	}
	ch, err := c.StartDownload(ctx)
	if err != nil {
		return Throughput{}, fmt.Errorf("start ndt7 download: %w", err)
	}
	return consume(ctx, ch, true, true)
}

// Upload measures upload goodput.
func (m *MLabClient) Upload(ctx context.Context, serviceURL string) (Throughput, error) {
	c, err := newLibClient(serviceURL)
	if err != nil {
		return Throughput{}, err
	}
	ch, err := c.StartUpload(ctx)
	if err != nil {
		return Throughput{}, fmt.Errorf("start ndt7 upload: %w", err)
	}
	return consume(ctx, ch, false, false)
}

// consume drains a measurement channel, computing goodput from the last
// application-level sample and (for downloads) RTT statistics from TCPInfo.
//
// When partialOnDeadline is true and ctx is cancelled, it returns the statistics
// gathered so far (used by the latency-only path); otherwise a cancelled ctx
// returns ctx.Err() so the full-measurement path fails cleanly.
func consume(ctx context.Context, ch <-chan spec.Measurement, wantRTT, partialOnDeadline bool) (Throughput, error) {
	var (
		lastBytes, lastElapsed int64
		rtts                   []float64
	)
	for {
		select {
		case <-ctx.Done():
			if partialOnDeadline {
				return finalize(lastBytes, lastElapsed, rtts), nil
			}
			return Throughput{}, ctx.Err()
		case m, ok := <-ch:
			if !ok {
				return finalize(lastBytes, lastElapsed, rtts), nil
			}
			if m.AppInfo != nil && m.AppInfo.ElapsedTime > 0 {
				lastBytes = m.AppInfo.NumBytes
				lastElapsed = m.AppInfo.ElapsedTime
			}
			if wantRTT && m.TCPInfo != nil && m.TCPInfo.RTT > 0 {
				rtts = append(rtts, float64(m.TCPInfo.RTT)/1000.0) // µs -> ms
			}
		}
	}
}

func finalize(numBytes, elapsedUS int64, rtts []float64) Throughput {
	var t Throughput
	if elapsedUS > 0 {
		// bits / microseconds == megabits / second
		t.Mbps = float64(numBytes) * 8.0 / float64(elapsedUS)
	}
	t.MinRTTMs, t.JitterMs = rttStats(rtts)
	return t
}

func rttStats(rtts []float64) (minRTT, jitter float64) {
	if len(rtts) == 0 {
		return 0, 0
	}
	minRTT = rtts[0]
	var sum float64
	for _, v := range rtts {
		if v < minRTT {
			minRTT = v
		}
		sum += v
	}
	mean := sum / float64(len(rtts))
	var variance float64
	for _, v := range rtts {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(rtts))
	return minRTT, math.Sqrt(variance)
}
