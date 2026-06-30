//go:build integration

// Package integration exercises velox end-to-end against the live M-Lab network.
// Run explicitly: go test -race -tags=integration ./test/integration/...
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/koller-nexus/velox/internal/geo"
	"github.com/koller-nexus/velox/internal/locate"
	"github.com/koller-nexus/velox/internal/ndt7"
	"github.com/koller-nexus/velox/internal/speedtest"
)

func TestLocateReturnsServers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	servers, err := locate.NewClient().Nearest(ctx)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if len(servers) == 0 {
		t.Fatal("expected at least one server")
	}
}

func TestFullSpeedTestAutoDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	runner := speedtest.NewRunner(ndt7.NewMLabClient())
	res := runner.Run(ctx, locate.Server{Machine: "(auto)", IsFallback: true}, nil)

	if !res.Online {
		t.Fatal("expected online")
	}
	if res.DownloadMbps <= 0 {
		t.Errorf("download should be positive, got %v", res.DownloadMbps)
	}
	if res.UploadMbps <= 0 {
		t.Errorf("upload should be positive, got %v", res.UploadMbps)
	}
}

func TestGeoResolveLive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	est, err := geo.NewIPResolver("").Resolve(ctx)
	if err != nil {
		t.Fatalf("geo resolve: %v", err)
	}
	if est.Lat == 0 && est.Lon == 0 {
		t.Errorf("expected non-zero coordinates")
	}
}
