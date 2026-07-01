package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/koller-nexus/velox/internal/speedtest"
)

const pingUsage = `velox ping — measure latency and jitter only (no throughput).

USAGE:
  velox ping [--json] [--server <url>] [--timeout <dur>]

Samples round-trip time over a short window and reports latency and jitter,
skipping the download and upload phases. Selects a server like the full test
(honoring consent) unless --server overrides it.
`

// pingCommand implements `velox ping` (FR-013): a latency-only quick check.
func (a *App) pingCommand() *Command {
	return &Command{
		Name:    "ping",
		Summary: "Measure latency and jitter only (no throughput)",
		Usage:   pingUsage,
		Run: func(ctx context.Context, args []string) int {
			fs := flag.NewFlagSet("ping", flag.ContinueOnError)
			asJSON := fs.Bool("json", false, "emit machine-readable JSON")
			server := fs.String("server", "", "override the test server (ndt7 service URL)")
			timeout := fs.Duration("timeout", 10*time.Second, "overall budget")
			if code, handled := a.parseCommandFlags(fs, pingUsage, args); handled {
				return code
			}

			ctx, cancel := context.WithTimeout(ctx, *timeout)
			defer cancel()

			srv, dist := a.selectServer(ctx, *server, false)
			res := a.Runner.Latency(ctx, srv, dist)

			if *asJSON {
				enc := json.NewEncoder(a.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					fmt.Fprintf(a.Stderr, "velox: render error: %v\n", err)
					return ExitFailure
				}
			} else {
				renderPingHuman(a.Stdout, res)
			}
			if !res.Online {
				return ExitFailure
			}
			return ExitOK
		},
	}
}

func renderPingHuman(w io.Writer, res speedtest.LatencyResult) {
	fmt.Fprintln(w, "velox ping")
	if !res.Online {
		fmt.Fprintln(w, "  Status:   offline (no connectivity)")
		return
	}
	if res.Server != nil {
		loc := res.Server.Machine
		if res.Server.City != "" {
			loc = fmt.Sprintf("%s (%s, %s)", res.Server.Machine, res.Server.City, res.Server.Country)
		}
		if res.DistanceKm != nil {
			fmt.Fprintf(w, "  Server:   %s — %.0f km\n", loc, *res.DistanceKm)
		} else {
			fmt.Fprintf(w, "  Server:   %s\n", loc)
		}
	}
	fmt.Fprintf(w, "  Latency:  %.1f ms   (jitter %.1f ms)\n", res.LatencyMs, res.JitterMs)
}
