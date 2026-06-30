package ndt7

import (
	"context"
	"math"
	"testing"

	"github.com/m-lab/ndt7-client-go/spec"
)

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestFinalizeComputesMbps(t *testing.T) {
	// 1_250_000 bytes in 1_000_000 µs => 10 Mbps.
	got := finalize(1_250_000, 1_000_000, nil)
	if !approx(got.Mbps, 10.0, 1e-9) {
		t.Errorf("Mbps = %v, want 10", got.Mbps)
	}
}

func TestRTTStats(t *testing.T) {
	min, jitter := rttStats([]float64{10, 20, 30})
	if min != 10 {
		t.Errorf("min = %v, want 10", min)
	}
	// stddev of {10,20,30} = sqrt(200/3) ≈ 8.165
	if !approx(jitter, math.Sqrt(200.0/3.0), 1e-6) {
		t.Errorf("jitter = %v, want ~8.165", jitter)
	}
	if m, j := rttStats(nil); m != 0 || j != 0 {
		t.Errorf("empty rtts should yield 0,0 got %v,%v", m, j)
	}
}

func TestConsumeDrainsChannel(t *testing.T) {
	ch := make(chan spec.Measurement, 3)
	ch <- spec.Measurement{
		AppInfo: &spec.AppInfo{NumBytes: 500_000, ElapsedTime: 500_000},
		TCPInfo: &spec.TCPInfo{},
	}
	// Last AppInfo wins: 2_500_000 bytes in 1_000_000 µs => 20 Mbps.
	last := spec.Measurement{AppInfo: &spec.AppInfo{NumBytes: 2_500_000, ElapsedTime: 1_000_000}}
	last.TCPInfo = &spec.TCPInfo{}
	last.TCPInfo.RTT = 15_000 // 15 ms
	ch <- last
	close(ch)

	got, err := consume(context.Background(), ch, true)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if !approx(got.Mbps, 20.0, 1e-9) {
		t.Errorf("Mbps = %v, want 20", got.Mbps)
	}
	if !approx(got.MinRTTMs, 15.0, 1e-9) {
		t.Errorf("MinRTTMs = %v, want 15", got.MinRTTMs)
	}
}

func TestConsumeHonorsCancellation(t *testing.T) {
	ch := make(chan spec.Measurement) // never sends
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := consume(ctx, ch, false); err == nil {
		t.Fatal("expected cancellation error")
	}
}
