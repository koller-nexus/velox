# velox

A fast, privacy-respecting command-line internet speed test, in the spirit of
`speedtest-cli`. velox measures **connectivity, latency, jitter, download, and
upload** against the nearest open [M-Lab](https://www.measurementlab.net/)
test server — with no API key, no elevated privileges, and an explicit consent
gate before it ever uses your location.

[![CI](https://github.com/koller-nexus/velox/actions/workflows/ci.yml/badge.svg)](https://github.com/koller-nexus/velox/actions/workflows/ci.yml)
![Go 1.26](https://img.shields.io/badge/go-1.26.4-00ADD8)
![License: MIT](https://img.shields.io/badge/license-MIT-green)

## Install

```bash
go install github.com/koller-nexus/velox/cmd/velox@latest
# or build locally:
make build && ./bin/velox --version
```

Pre-built static binaries for Linux, macOS, and Windows (amd64 + arm64) are
produced by `make cross` (output in `dist/`).

## Usage

```bash
velox --check-internet           # run a full speed test
velox --check-internet --json    # machine-readable output
velox --server <wss-url>         # test against a specific ndt7 server
velox --version
velox --help
```

Example:

```text
Velox speed test
  Server:    mlab2-fln01 (Florianopolis, BR) — 12 km
  Latency:   7.8 ms   (jitter 1.2 ms)
  Download:  347.2 Mbps
  Upload:    322.5 Mbps
```

### Metrics

| Metric            | Definition                                             |
| ----------------- | ------------------------------------------------------ |
| Latency           | Minimum round-trip time (ms), sampled during download. |
| Jitter            | RTT variation (ms) over the sampling window.           |
| Download / Upload | Mean application-layer goodput (Mbps).                 |

### Exit codes

`0` success · `1` measurement/network failure (offline, phase failed) · `2` usage error.

## Privacy & consent

To pick the **nearest** server, velox needs your approximate location. It derives
a coarse, city-level estimate from your **public IP** via an HTTPS geolocation
lookup — and it asks for your consent **once** before doing so:

```bash
velox consent --status     # granted | denied | unset
velox consent --grant      # allow location lookups
velox consent --deny       # disallow
velox consent --reset      # forget the decision (asked again next time)
```

- Your decision is stored locally under your OS config directory.
- If you **decline** — or run non-interactively (CI, cron, pipes) — velox never
  performs a location lookup and falls back to a default/auto-discovered server;
  no client→server distance is shown.
- The location estimate is held in memory only and never written to disk.

## Development

```bash
make all          # gofmt check + vet + lint + race tests
make test         # go test -race ./...
make security     # gosec + govulncheck (fails at HIGH+)
make build        # static binary -> bin/velox
go test -race -tags=integration ./test/integration/...   # opt-in, live M-Lab
```

velox follows a documented [constitution](.specify/memory/constitution.md):
clean idiomatic Go, CLI-first UX, test-first development, measurement accuracy,
and minimal dependencies (a single runtime dependency — the canonical M-Lab
ndt7 client).

## License

[MIT](LICENSE) © 2026 William Koller
