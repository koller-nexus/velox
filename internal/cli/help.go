package cli

import "fmt"

// usage is printed for --help and bare invocation. It performs no network or
// consent work (FR-016).
const usage = `velox — measure your internet speed against the nearest open test server.

USAGE:
  velox --check-internet [flags]
  velox consent <--status|--reset|--grant|--deny>
  velox --version
  velox --help

FLAGS (--check-internet):
  --json            Emit machine-readable JSON (units: latency/jitter in ms, throughput in Mbps).
  --server <url>    Override the test server (ndt7 service URL); bypasses selection.
  --timeout <dur>   Overall run budget (default 60s).
  --no-progress     Disable the loading indicator (also disabled when not a TTY or under --verbose).
  -v, --verbose     Verbose diagnostics on stderr.

METRICS:
  Latency  = minimum round-trip time (ms) sampled during the download phase.
  Jitter   = round-trip time variation (ms) over the sampling window.
  Download/Upload = mean application-layer goodput (Mbps).

PRIVACY:
  Picking the nearest server uses your approximate, IP-based location. velox asks
  for consent once before any location lookup and remembers your choice. Manage it
  with 'velox consent'. Declining still runs the test against a fallback server.`

func (a *App) printUsage() {
	fmt.Fprintln(a.Stdout, usage)
}
