# Changelog

All notable changes to velox are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Command help system: `velox help` prints an overview of every command
  (generated from a command registry, so it stays complete), and
  `velox help <command>` / `velox <command> --help` print detailed usage.
  Unknown commands or flags report to stderr, suggest `velox help`, and exit `2`.
- `velox version` subcommand — the command form of `--version` (offline).
- `velox servers` — list the nearest test servers velox would select (the pick
  plus ~4 nearest alternatives) with location and distance, honoring the consent
  gate; `--json` supported.
- `velox ping` — measure latency and jitter only, sampling round-trip time over a
  short (~5s) window via the ndt7 download subtest (no download/upload
  throughput); `--json` and `--server`/`--timeout` supported.
- `velox config` — show the config file path/directory, consent decision, and
  effective settings; read-only and offline; `--json` supported.
- Loading indicator for `velox --check-internet`: on an interactive terminal an
  animated spinner shows the current phase (selecting server, checking
  connectivity, measuring download, measuring upload) and elapsed time on stderr,
  then clears before results print. It is suppressed on non-terminals (pipe,
  redirect, CI, `TERM=dumb`, `NO_COLOR`) and under `--verbose`, keeping stdout and
  `--json` output clean.
- `--no-progress` flag to disable the loading indicator even on a terminal.
- `velox --check-internet`: full internet speed test — connectivity, latency,
  jitter, download and upload throughput — against the nearest open test server.
- Nearest-server selection via the M-Lab Locate API v2 (open registry, no API
  key) ranked by geographic distance using a bundled metro-coordinate table.
- Consent-gated, IP-based geolocation over HTTPS to compute client→server
  distance; consent is asked once, persisted, and revocable (`velox consent`).
- Safe non-interactive default: when no TTY is present, location is never
  requested and a fallback/auto-discovered server is used.
- `--json` machine-readable output, `-v/--verbose` diagnostics on stderr, and
  meaningful exit codes (0 success, 1 measurement/network failure, 2 usage).
- ndt7 measurement over WebSocket-over-TLS — runs without elevated privileges.
- Security gate (`make security`): gosec + govulncheck, failing CI at HIGH+.
- Cross-compiled static binaries for Linux, macOS, and Windows (amd64 + arm64).

[Unreleased]: https://github.com/koller-nexus/velox/commits/main
