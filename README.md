# velox

A fast, privacy-respecting command-line internet speed test, in the spirit of
`speedtest-cli`. velox measures **connectivity, latency, jitter, download, and
upload** against the nearest open [M-Lab](https://www.measurementlab.net/)
test server — with no API key, no elevated privileges, and an explicit consent
gate before it ever uses your location.

[![CI](https://github.com/koller-nexus/velox/actions/workflows/ci.yml/badge.svg)](https://github.com/koller-nexus/velox/actions/workflows/ci.yml)
![Go 1.26](https://img.shields.io/badge/go-1.26.4-00ADD8)
![License: MIT](https://img.shields.io/badge/license-MIT-green)

## Installation

### macOS and Linux (one-command install)

```bash
curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh
```

The installer detects your OS and architecture, downloads the correct archive,
verifies the SHA256 checksum, and installs `velox` to `/usr/local/bin` (or
`~/.local/bin` if you don't have write permission). Make sure the chosen
directory is on your `PATH`.

You can override the version or install directory:

```bash
VERSION=v1.2.3 INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh
```

### macOS (Homebrew)

```bash
brew tap koller-nexus/tap
brew install velox
```

### Windows

Download the latest `.zip` for your architecture from the
[Releases](https://github.com/koller-nexus/velox/releases) page, extract it, and
run `velox.exe`. No external runtime is required.

### Fallback: install with Go

If you already have Go installed:

```bash
go install github.com/koller-nexus/velox/cmd/velox@latest
```

### Build from source

```bash
make build && ./bin/velox --version
```

Pre-built static binaries for Linux, macOS, and Windows are produced
automatically on every release tag by GoReleaser.

## Usage

```bash
velox --check-internet                       # run a full speed test
velox --check-internet --json                # machine-readable output
velox --check-internet --no-progress         # disable the loading indicator
velox --check-internet --nearest-provider    # target the server nearest to your closest ISP POP
velox --server <wss-url>                     # test against a specific ndt7 server
velox --version
velox --help
```

### Commands

Beyond the full test, velox exposes small, focused subcommands. Run
`velox help` for the overview, or `velox help <command>` (same as
`velox <command> --help`) for details.

```bash
velox help [command]   # overview, or detailed help for a command
velox version          # print the version (same as --version)
velox servers          # list the nearest servers velox would use (--json)
velox ping             # latency + jitter only, no throughput (--json)
velox config           # show config path and effective settings (--json)
velox consent --status # manage location-lookup consent
```

`help`, `version`, and `config` never touch the network. `servers` uses your
location only with consent (falling back to registry order otherwise); `ping`
samples round-trip time over a short (~5s) window against the selected server.
Unknown commands or flags print an error to stderr and exit with code `2`.


Example:

```text
Velox speed test
  Server:           mlab2-fln01 (Florianopolis, BR) — 12 km
  Nearest provider: Vivo (São Paulo — Centro) — 2.3 km
  Latency:          7.8 ms   (jitter 1.2 ms)
  Download:         347.2 Mbps
  Upload:           322.5 Mbps
```

With `--nearest-provider`, velox selects the M-Lab test server closest to the
POP of the provider nearest to you. The nearest-provider line is shown whenever
location consent is granted and a match exists in the bundled provider catalog;
it is omitted (with a silent fallback to default server selection) when consent
is denied, the lookup fails, or no provider POP is nearby.

On an interactive terminal, velox shows an animated loading indicator on stderr
while the test runs (selecting server → checking connectivity → measuring
download → measuring upload, with elapsed time), then clears it before printing
results. The indicator is written only to stderr, so `--json` and piped output on
stdout stay clean. It is suppressed automatically when output is not a terminal
(pipe, redirect, CI, `TERM=dumb`, or `NO_COLOR`), under `--verbose`, or with
`--no-progress`.

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
  no client→server distance or nearest-provider metadata is shown.
- The location estimate and nearest-provider result are held in memory only and
  never written to disk.

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
