package speedtest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/koller-nexus/velox/internal/provider"
)

func TestMeasurementResultJSONShape(t *testing.T) {
	res := MeasurementResult{
		Online:       true,
		LatencyMs:    8.4,
		DownloadMbps: 240,
		UploadMbps:   38.7,
		Server:       &ServerInfo{Machine: "m", City: "São Paulo", Country: "BR"},
		StartedAt:    time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
		DurationMs:   1234,
		PhaseStatus: map[Phase]PhaseOutcome{
			PhaseConnectivity: {OK: true},
			PhaseDownload:     {OK: true, Value: 240},
		},
	}
	res2 := res
	res2.NearestProvider = &provider.Result{
		Provider:   provider.Provider{Name: "Vivo"},
		POP:        provider.POP{Label: "São Paulo — Centro", City: "São Paulo", Country: "BR"},
		DistanceKm: 2.3,
	}

	for _, tc := range []struct {
		name string
		res  MeasurementResult
	}{
		{"without nearest provider", res},
		{"with nearest provider", res2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.res)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			// Required keys per contracts/result.schema.json.
			for _, k := range []string{"online", "startedAt", "durationMs", "phaseStatus", "distanceKm"} {
				if _, ok := m[k]; !ok {
					t.Errorf("missing required key %q in JSON output", k)
				}
			}
			// distanceKm must be present and null when unset.
			if m["distanceKm"] != nil {
				t.Errorf("distanceKm should be null when unset, got %v", m["distanceKm"])
			}
		})
	}
}

func TestMeasurementResultJSON_NearestProvider(t *testing.T) {
	res := MeasurementResult{
		Online: true,
		NearestProvider: &provider.Result{
			Provider:   provider.Provider{Name: "Vivo"},
			POP:        provider.POP{Label: "São Paulo — Centro", City: "São Paulo", Country: "BR", Lat: -23.55, Lon: -46.63},
			DistanceKm: 2.3,
		},
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	np, ok := m["nearestProvider"].(map[string]any)
	if !ok {
		t.Fatalf("nearestProvider missing or wrong type: %v", m["nearestProvider"])
	}
	if np["distanceKm"] != 2.3 {
		t.Errorf("distanceKm = %v, want 2.3", np["distanceKm"])
	}
}
