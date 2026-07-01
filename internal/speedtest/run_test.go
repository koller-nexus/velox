package speedtest

import (
	"context"
	"errors"
	"testing"

	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/ndt7"
)

// fakeClient implements ndt7.Client without any network.
type fakeClient struct {
	dl, ul ndt7.Throughput
	dlErr  error
	ulErr  error
	calls  []string
}

func (f *fakeClient) Download(_ context.Context, _ string) (ndt7.Throughput, error) {
	f.calls = append(f.calls, "download")
	return f.dl, f.dlErr
}

func (f *fakeClient) Upload(_ context.Context, _ string) (ndt7.Throughput, error) {
	f.calls = append(f.calls, "upload")
	return f.ul, f.ulErr
}

func newTestRunner(c ndt7.Client, dialErr error) *Runner {
	return &Runner{
		Client: c,
		Dial:   func(context.Context, string) error { return dialErr },
	}
}

func TestRunHappyPath(t *testing.T) {
	c := &fakeClient{
		dl: ndt7.Throughput{Mbps: 240, MinRTTMs: 8.4, JitterMs: 1.2},
		ul: ndt7.Throughput{Mbps: 38.7},
	}
	r := newTestRunner(c, nil)
	dist := 12.0
	res := r.Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, &dist, nil)

	if !res.Online {
		t.Fatal("expected online")
	}
	if res.DownloadMbps != 240 || res.UploadMbps != 38.7 {
		t.Errorf("throughput wrong: %+v", res)
	}
	if res.LatencyMs != 8.4 || res.JitterMs != 1.2 {
		t.Errorf("latency/jitter wrong: %+v", res)
	}
	for _, p := range []Phase{PhaseConnectivity, PhaseLatency, PhaseDownload, PhaseUpload} {
		if !res.PhaseStatus[p].OK {
			t.Errorf("phase %s not OK", p)
		}
	}
	if res.DistanceKm == nil || *res.DistanceKm != 12.0 {
		t.Errorf("distance not carried through")
	}
}

func TestRunOfflineSkipsThroughput(t *testing.T) {
	c := &fakeClient{}
	r := newTestRunner(c, errors.New("network down"))
	res := r.Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, nil, nil)

	if res.Online {
		t.Fatal("expected offline")
	}
	if res.PhaseStatus[PhaseConnectivity].OK {
		t.Error("connectivity should have failed")
	}
	if len(c.calls) != 0 {
		t.Errorf("throughput phases must be skipped when offline, got %v", c.calls)
	}
}

func TestRunPhaseFailureRecorded(t *testing.T) {
	c := &fakeClient{dlErr: errors.New("download stalled"), ul: ndt7.Throughput{Mbps: 5}}
	r := newTestRunner(c, nil)
	res := r.Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, nil, nil)

	if !res.Online {
		t.Fatal("expected online (connectivity ok)")
	}
	if res.PhaseStatus[PhaseDownload].OK {
		t.Error("download phase should be marked failed")
	}
	if res.PhaseStatus[PhaseDownload].Error == "" {
		t.Error("failed phase should record an error")
	}
	if !res.PhaseStatus[PhaseUpload].OK {
		t.Error("upload should still run after download failure")
	}
}

// recordingReporter captures the order of phase-start notifications.
type recordingReporter struct{ phases []Phase }

func (r *recordingReporter) Phase(p Phase) { r.phases = append(r.phases, p) }

func TestRunReportsPhasesInOrder(t *testing.T) {
	t.Run("healthy run announces connectivity, download, upload", func(t *testing.T) {
		c := &fakeClient{dl: ndt7.Throughput{Mbps: 100}, ul: ndt7.Throughput{Mbps: 10}}
		rep := &recordingReporter{}
		newTestRunner(c, nil).Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, nil, rep)

		want := []Phase{PhaseConnectivity, PhaseDownload, PhaseUpload}
		if !equalPhases(rep.phases, want) {
			t.Errorf("phases = %v, want %v", rep.phases, want)
		}
	})

	t.Run("offline run announces only connectivity", func(t *testing.T) {
		rep := &recordingReporter{}
		newTestRunner(&fakeClient{}, errors.New("down")).Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, nil, rep)

		want := []Phase{PhaseConnectivity}
		if !equalPhases(rep.phases, want) {
			t.Errorf("phases = %v, want %v", rep.phases, want)
		}
	})

	t.Run("nil reporter is safe", func(_ *testing.T) {
		c := &fakeClient{dl: ndt7.Throughput{Mbps: 100}, ul: ndt7.Throughput{Mbps: 10}}
		// Must not panic.
		newTestRunner(c, nil).Run(context.Background(), locate.Server{Machine: "m", DownloadURL: "wss://h/d"}, nil, nil)
	})
}

func equalPhases(a, b []Phase) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
