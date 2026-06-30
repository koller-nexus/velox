package speedtest

import (
	"encoding/json"
	"testing"
	"time"
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
	data, err := json.Marshal(res)
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
}
