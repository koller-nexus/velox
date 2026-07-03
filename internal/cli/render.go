package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/koller-nexus/velox/internal/speedtest"
)

// renderJSON writes the result as a single JSON object (contracts/result.schema.json).
func renderJSON(w io.Writer, res speedtest.MeasurementResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

// renderHuman writes a human-readable summary to w.
func renderHuman(w io.Writer, res speedtest.MeasurementResult) {
	if !res.Online {
		fmt.Fprintln(w, "Velox speed test")
		fmt.Fprintln(w, "  Status:    offline (no connectivity)")
		return
	}

	header := "Velox speed test"
	if res.Server != nil && res.Server.IsFallback {
		header += "  (location disabled — using fallback server)"
	}
	fmt.Fprintln(w, header)

	if res.Server != nil {
		loc := res.Server.Machine
		if res.Server.City != "" {
			loc = fmt.Sprintf("%s (%s, %s)", res.Server.Machine, res.Server.City, res.Server.Country)
		}
		if res.DistanceKm != nil {
			fmt.Fprintf(w, "  Server:    %s — %.0f km", loc, *res.DistanceKm)
		} else {
			fmt.Fprintf(w, "  Server:    %s", loc)
		}
		if res.NearestProvider != nil && res.NearestProvider.SelectedServer != nil {
			fmt.Fprint(w, "  [nearest-provider target]")
		}
		fmt.Fprintln(w)
	}
	if res.NearestProvider != nil {
		np := res.NearestProvider
		fmt.Fprintf(w, "  Nearest provider: %s (%s) — %.1f km\n", np.Provider.Name, np.POP.Label, np.DistanceKm)
	}
	fmt.Fprintf(w, "  Latency:   %.1f ms   (jitter %.1f ms)\n", res.LatencyMs, res.JitterMs)
	fmt.Fprintf(w, "  Download:  %.1f Mbps\n", res.DownloadMbps)
	fmt.Fprintf(w, "  Upload:    %.1f Mbps\n", res.UploadMbps)
}
