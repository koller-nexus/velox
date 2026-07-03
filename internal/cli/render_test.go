package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/koller-nexus/velox/internal/provider"
	"github.com/koller-nexus/velox/internal/speedtest"
)

func TestRenderHuman_IncludesNearestProvider(t *testing.T) {
	res := speedtest.MeasurementResult{
		Online:       true,
		LatencyMs:    7.8,
		JitterMs:     1.2,
		DownloadMbps: 347.2,
		UploadMbps:   322.5,
		Server:       &speedtest.ServerInfo{Machine: "mlab2-gru01", City: "São Paulo", Country: "BR"},
		NearestProvider: &provider.Result{
			Provider:   provider.Provider{Name: "Vivo"},
			POP:        provider.POP{Label: "São Paulo — Centro", City: "São Paulo", Country: "BR"},
			DistanceKm: 2.3,
		},
	}

	var out bytes.Buffer
	renderHuman(&out, res)
	got := out.String()

	if !strings.Contains(got, "Nearest provider:") {
		t.Errorf("human output missing nearest-provider line:\n%s", got)
	}
	if !strings.Contains(got, "Vivo") {
		t.Errorf("human output missing provider name:\n%s", got)
	}
	if !strings.Contains(got, "São Paulo — Centro") {
		t.Errorf("human output missing POP label:\n%s", got)
	}
	if !strings.Contains(got, "2.3 km") {
		t.Errorf("human output missing distance:\n%s", got)
	}
}

func TestRenderHuman_OmitsNearestProviderWhenNil(t *testing.T) {
	res := speedtest.MeasurementResult{
		Online: true,
		Server: &speedtest.ServerInfo{Machine: "mlab2-gru01"},
	}

	var out bytes.Buffer
	renderHuman(&out, res)
	got := out.String()

	if strings.Contains(got, "Nearest provider:") {
		t.Errorf("human output should not contain nearest-provider line:\n%s", got)
	}
}
